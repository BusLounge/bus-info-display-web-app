from fastapi import FastAPI, HTTPException
from pydantic import BaseModel
import pandas as pd
import numpy as np
import joblib
import xgboost as xgb
import psycopg2
from psycopg2.extras import RealDictCursor
import os
from datetime import datetime
import json

app = FastAPI(title="ETA Engine ML Service", version="1.0.1", description="Service for predicting bus arrival times")

xgb_model = None
label_encoders = None
driver_ratios = None
bus_ratios = None
lounge_metrics = None

FEATURES = [
    'distance_km',
    'baseline_duration_minutes',
    'baseline_speed_kmh',
    'segment_order',
    'driver_id_enc',
    'bus_id_enc',
    'driver_duration_ratio',
    'bus_duration_ratio',
    'time_of_day_category_enc',
    'day_of_week_enc',
    'departure_hour',
    'segment_hour',
    'is_weekend',
    'is_holiday',
    'total_passengers',
    'weather_condition_enc',
    'traffic_level_enc',
    'road_type_enc'
]

def load_models():
    global xgb_model, label_encoders, driver_ratios, bus_ratios, lounge_metrics
    try:
        xgb_model = xgb.XGBRegressor()
        xgb_model.load_model("eta_model.json")
        
        label_encoders = joblib.load("label_encoders.pkl")
        
        driver_ratios = pd.read_csv("driver_duration_ratios.csv")
        bus_ratios = pd.read_csv("bus_duration_ratios.csv")
        
        try:
            lounge_metrics = pd.read_csv("lounge_stop_metrics_clean.csv")
        except FileNotFoundError:
            print("Warning: lounge_stop_metrics_clean.csv not found, using default dwell times.")
            lounge_metrics = None
            
        print("Models loaded successfully.")
    except Exception as e:
        print(f"Error loading models: {e}")

@app.on_event("startup")
async def startup_event():
    load_models()

class ETARequest(BaseModel):
    master_route_id: str
    lounge_id: str
    departure_datetime: str  # ISO format
    driver_id: str
    bus_id: str
    weather_condition: str = 'clear'
    traffic_level: str = 'moderate'
    total_passengers: int = 30
    is_holiday: bool = False
    
    db_dsn: str = "postgresql://postgres:postgres@localhost:5432/businfo"

def get_segments_to_lounge(dsn: str, master_route_id: str, lounge_id: str):
    conn = psycopg2.connect(dsn)
    cursor = conn.cursor(cursor_factory=RealDictCursor)
    
    query = """
    WITH TargetLounge AS (
        SELECT lr.stop_before_id
        FROM lounge_routes lr
        WHERE lr.master_route_id = %s AND lr.lounge_id = %s
        LIMIT 1
    )
    SELECT rs.*
    FROM route_segments rs
    WHERE rs.master_route_id = %s
    AND rs.segment_order <= (
        SELECT segment_order
        FROM route_segments 
        WHERE master_route_id = %s AND end_point_id = (SELECT stop_before_id FROM TargetLounge)
        LIMIT 1
    )
    ORDER BY rs.segment_order ASC
    """
    cursor.execute(query, (master_route_id, lounge_id, master_route_id, master_route_id))
    segments = cursor.fetchall()
    
    if not segments:
        query_simple = "SELECT * FROM route_segments WHERE master_route_id = %s ORDER BY segment_order ASC"
        cursor.execute(query_simple, (master_route_id,))
        all_segments = cursor.fetchall()
        
        cursor.execute("SELECT stop_before_id FROM lounge_routes WHERE master_route_id = %s AND lounge_id = %s", (master_route_id, lounge_id))
        res = cursor.fetchone()
        
        if res:
            stop_before = res['stop_before_id']
            target_order = 9999
            for s in all_segments:
                if s['end_point_id'] == stop_before:
                    target_order = s['segment_order']
                    break
            segments = [s for s in all_segments if s['segment_order'] <= target_order]
            
    cursor.close()
    conn.close()
    
    return segments

def get_driver_ratio(driver_id: str):
    if driver_ratios is None:
        return 1.0
    row = driver_ratios[driver_ratios['driver_id'] == driver_id]
    if not row.empty:
        return float(row.iloc[0]['driver_duration_ratio'])
    return 1.0

def get_bus_ratio(bus_id: str):
    if bus_ratios is None:
        return 1.0
    row = bus_ratios[bus_ratios['bus_id'] == bus_id]
    if not row.empty:
        return float(row.iloc[0]['bus_duration_ratio'])
    return 1.0

def predict_dwell_time(lounge_id, master_route_id, is_peak_hour):
    if lounge_metrics is None:
        return 15.0
    
    row = lounge_metrics[
        (lounge_metrics['lounge_id'] == lounge_id) &
        (lounge_metrics['master_route_id'] == master_route_id)
    ]
    if row.empty:
        return 15.0
        
    if is_peak_hour:
        return float(row.iloc[0]['peak_hour_dwell_time_minutes'])
    else:
        return float(row.iloc[0]['off_peak_dwell_time_minutes'])

@app.post("/api/v1/predict")
async def predict_eta(req: ETARequest):
    try:
        segments = get_segments_to_lounge(req.db_dsn, req.master_route_id, req.lounge_id)
        if not segments:
            raise HTTPException(status_code=404, detail="Segments to lounge not found")
            
        departure_dt = pd.to_datetime(req.departure_datetime)
        hour = departure_dt.hour
        is_weekend = departure_dt.weekday() >= 5
        day_of_week = departure_dt.strftime('%A').lower()
        
        tod_map = {
            range(5, 8): 'early_morning',
            range(8, 10): 'morning_peak',
            range(10, 16): 'midday',
            range(16, 20): 'evening_peak',
        }
        time_of_day = 'night'
        for r, label in tod_map.items():
            if hour in r:
                time_of_day = label
                break
                
        is_peak_hour = time_of_day in ['morning_peak', 'evening_peak']
        
        total_minutes = 0.0
        
        d_ratio = get_driver_ratio(req.driver_id)
        b_ratio = get_bus_ratio(req.bus_id)
        
        for seg in segments:
            cumulative_hours = total_minutes / 60
            seg_hour = (departure_dt + pd.Timedelta(hours=cumulative_hours)).hour
            
            feature_row = {
                'distance_km': float(seg.get('distance_km', 0)),
                'baseline_duration_minutes': float(seg.get('baseline_duration_minutes', 0)),
                'baseline_speed_kmh': float(seg.get('baseline_speed_kmh', 40)),
                'segment_order': int(seg.get('segment_order', 1)),
                'driver_duration_ratio': d_ratio,
                'bus_duration_ratio': b_ratio,
                'departure_hour': hour,
                'segment_hour': seg_hour,
                'is_weekend': int(is_weekend),
                'is_holiday': int(req.is_holiday),
                'total_passengers': req.total_passengers,
            }
            
            cat_values = {
                'driver_id': req.driver_id,
                'bus_id': req.bus_id,
                'time_of_day_category': time_of_day,
                'day_of_week': day_of_week,
                'weather_condition': req.weather_condition,
                'traffic_level': req.traffic_level,
                'road_type': str(seg.get('road_type', 'unknown')),
            }
            
            for col, val in cat_values.items():
                enc_col = col + '_enc'
                if enc_col in FEATURES and col in label_encoders:
                    le = label_encoders[col]
                    val_str = str(val)
                    if val_str in le.classes_:
                        feature_row[enc_col] = float(le.transform([val_str])[0])
                    else:
                        feature_row[enc_col] = -1.0
                else:
                    feature_row[enc_col] = -1.0
                    
            row_array = np.array([[feature_row.get(f, 0) for f in FEATURES]])
            pred_duration = float(xgb_model.predict(row_array)[0])
            pred_duration = max(pred_duration, 0.1)
            
            total_minutes += pred_duration
            
        eta = departure_dt + pd.Timedelta(minutes=total_minutes)
        dwell = predict_dwell_time(req.lounge_id, req.master_route_id, is_peak_hour)
        etd = eta + pd.Timedelta(minutes=dwell)
        
        return {
            "master_route_id": req.master_route_id,
            "lounge_id": req.lounge_id,
            "eta": eta.isoformat(),
            "etd": etd.isoformat(),
            "eta_minutes_from_departure": round(total_minutes, 1),
            "predicted_dwell_minutes": round(dwell, 1),
            "segments_traversed": len(segments),
            "driver_ratio": d_ratio,
            "bus_ratio": b_ratio
        }
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

from typing import List

class BatchETARequest(BaseModel):
    requests: List[ETARequest]

@app.post("/api/v1/predict_batch")
async def predict_eta_batch(batch_req: BatchETARequest):
    results = []
    # In a real app we'd parallelize this or batch predict through XGBoost
    # For now, process sequentially
    for req in batch_req.requests:
        try:
            res = await predict_eta(req)
            # Tag with a trip_id or identifier if provided, but we can return in same order
            results.append({"status": "success", "data": res})
        except Exception as e:
            results.append({"status": "error", "error": str(e)})
    return results
