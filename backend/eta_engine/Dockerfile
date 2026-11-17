FROM python:3.12-slim
# libgomp1 is required by xgboost.
RUN apt-get update \
    && apt-get install -y --no-install-recommends libgomp1 \
    && rm -rf /var/lib/apt/lists/* \
    && useradd -m -u 10001 app
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
# main.py loads the model files (eta_model.json, label_encoders.pkl,
# driver_duration_ratios.csv, bus_duration_ratios.csv) from this directory.
COPY . .
USER app
EXPOSE 8000
CMD ["uvicorn", "main:app", "--host", "0.0.0.0", "--port", "8000"]
