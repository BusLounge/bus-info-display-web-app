package main

import (
	"bus-schedule-lounge/internal/config"
	"bus-schedule-lounge/internal/database"
	"bus-schedule-lounge/internal/handlers"
	"bus-schedule-lounge/internal/middleware"
	"bus-schedule-lounge/internal/services"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	cfg := config.Load()

	db, err := database.NewDatabase(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	adRepo := database.NewAdvertisementRepository(db.DB)
	adService := services.NewAdvertisementService(adRepo)
	if err := adService.EnsureCalculationSchema(); err != nil {
		log.Fatal("Failed to initialize advertisement calculation schema:", err)
	}
	adHandler := handlers.NewAdvertisementHandler(adService, cfg.MediaDir)

	// Arrival Management
	arrivalRepo := database.NewArrivalRepository(db.DB)
	arrivalService := services.NewArrivalService(arrivalRepo)
	arrivalHandler := handlers.NewArrivalHandler(arrivalService)

	// Departure Management
	departureRepo := database.NewDepartureRepository(db.DB)
	departureService := services.NewDepartureService(departureRepo)
	departureHandler := handlers.NewDepartureHandler(departureService)

	// Lounge Management
	loungeRepo := database.NewLoungeRepository(db.DB)
	loungeService := services.NewLoungeService(loungeRepo)
	loungeHandler := handlers.NewLoungeHandler(loungeService)

	// Route Management
	routeRepo := database.NewRouteRepository(db.DB)
	routeService := services.NewRouteService(routeRepo)
	routeHandler := handlers.NewRouteHandler(routeService)

	// Broadcast Messaging
	broadcastRepo := database.NewBroadcastMessageRepository(db.DB)
	broadcastService := services.NewBroadcastMessageService(broadcastRepo)
	if err := broadcastService.EnsureSchema(); err != nil {
		log.Fatal("Failed to initialize broadcast message schema:", err)
	}
	broadcastHandler := handlers.NewBroadcastMessageHandler(broadcastService)
	authHandler := handlers.NewAuthHandler(
		cfg.JWTSecret,
		cfg.JWTRefreshSecret,
		cfg.AdminUser,
		cfg.AdminPass,
		cfg.JWTAccessTokenExpiry,
		cfg.JWTRefreshTokenExpiry,
	)

	// Lounge-specific Advertisements
	loungeAdRepo := database.NewLoungeAdRepository(db.DB)
	loungeAdService := services.NewLoungeAdService(loungeAdRepo)
	if err := loungeAdService.EnsureSchema(); err != nil {
		log.Fatal("Failed to initialize lounge ad schema:", err)
	}
	loungeAdHandler := handlers.NewLoungeAdHandler(loungeAdService)

	// Dashboard
	dashboardRepo := database.NewDashboardRepository(db.DB)
	dashboardService := services.NewDashboardService(dashboardRepo)
	dashboardHandler := handlers.NewDashboardHandler(dashboardService)

	router := mux.NewRouter()

	// Add CORS middleware first (before routes)
	router.Use(middleware.CORS)

	api := router.PathPrefix("/api").Subrouter()
	protected := api.PathPrefix("").Subrouter()
	protected.Use(middleware.RequireAuth(cfg.JWTSecret))

	api.HandleFunc("/auth/login", authHandler.Login).Methods("POST", "OPTIONS")
	api.HandleFunc("/auth/me", authHandler.Me).Methods("GET", "OPTIONS")

	protected.HandleFunc("/advertisements/upload-media", adHandler.UploadMedia).Methods("POST", "OPTIONS")
	api.HandleFunc("/advertisements", adHandler.GetAllAdvertisements).Methods("GET", "OPTIONS")
	protected.HandleFunc("/advertisements", adHandler.CreateAdvertisement).Methods("POST", "OPTIONS")
	api.HandleFunc("/advertisements/{id}", adHandler.GetAdvertisementByID).Methods("GET", "OPTIONS")
	protected.HandleFunc("/advertisements/{id}", adHandler.UpdateAdvertisement).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/advertisements/{id}", adHandler.DeleteAdvertisement).Methods("DELETE", "OPTIONS")
	protected.HandleFunc("/advertisements/conflicts", adHandler.CheckConflict).Methods("POST", "OPTIONS")
	api.HandleFunc("/tv/ads/{loungeGroup}", adHandler.GetTVAdsManifest).Methods("GET", "OPTIONS")
	api.HandleFunc("/advertisement-calculation/rates", adHandler.GetCalculationRates).Methods("GET", "OPTIONS")
	protected.HandleFunc("/advertisement-calculation/rates/{trafficLevel}", adHandler.UpsertCalculationRate).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/advertisement-calculation/logs", adHandler.RecordPlaybackLog).Methods("POST", "OPTIONS")
	api.HandleFunc("/advertisement-calculation/logs", adHandler.GetPlaybackLogs).Methods("GET", "OPTIONS")
	api.HandleFunc("/advertisement-calculation/report", adHandler.GetCostReport).Methods("GET", "OPTIONS")
	protected.HandleFunc("/advertisement-calculation/sync-scheduled", adHandler.SyncScheduledAdsCosts).Methods("POST", "OPTIONS")

	api.HandleFunc("/advertisement-groups", adHandler.GetAllGroups).Methods("GET", "OPTIONS")
	protected.HandleFunc("/advertisement-groups", adHandler.CreateGroup).Methods("POST", "OPTIONS")
	api.HandleFunc("/advertisement-groups/{id}", adHandler.GetGroupByID).Methods("GET", "OPTIONS")
	protected.HandleFunc("/advertisement-groups/{id}", adHandler.UpdateGroup).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/advertisement-groups/{id}", adHandler.DeleteGroup).Methods("DELETE", "OPTIONS")

	// Arrival Management routes
	api.HandleFunc("/arrivals", arrivalHandler.GetAllLoungeArrivals).Methods("GET", "OPTIONS")
	api.HandleFunc("/arrivals/lounge/{loungeId}", arrivalHandler.GetArrivalsByLoungeID).Methods("GET", "OPTIONS")

	// Departure Management routes
	api.HandleFunc("/departures", departureHandler.GetAllLoungeDepartures).Methods("GET", "OPTIONS")
	api.HandleFunc("/departures/lounge/{loungeId}", departureHandler.GetDeparturesByLoungeID).Methods("GET", "OPTIONS")

	// Lounge Management routes
	api.HandleFunc("/lounges", loungeHandler.GetAllLounges).Methods("GET", "OPTIONS")
	api.HandleFunc("/lounges/{id}", loungeHandler.GetLoungeByID).Methods("GET", "OPTIONS")
	api.HandleFunc("/lounges/{id}/route-segment-validation", loungeHandler.ValidateLoungeRouteSegments).Methods("GET", "OPTIONS")

	// Route Management routes
	api.HandleFunc("/routes", routeHandler.GetAllRoutes).Methods("GET", "OPTIONS")
	protected.HandleFunc("/routes", routeHandler.CreateRoute).Methods("POST", "OPTIONS")
	api.HandleFunc("/routes/{id}", routeHandler.GetRouteByID).Methods("GET", "OPTIONS")
	protected.HandleFunc("/routes/{id}", routeHandler.UpdateRoute).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/routes/{id}", routeHandler.DeleteRoute).Methods("DELETE", "OPTIONS")

	// Broadcast message routes
	api.HandleFunc("/broadcast-messages", broadcastHandler.GetAll).Methods("GET", "OPTIONS")
	protected.HandleFunc("/broadcast-messages", broadcastHandler.Create).Methods("POST", "OPTIONS")
	protected.HandleFunc("/broadcast-messages/{id}", broadcastHandler.Update).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/broadcast-messages/{id}", broadcastHandler.Delete).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/tv/broadcasts", broadcastHandler.GetActiveForTV).Methods("GET", "OPTIONS")

	// Lounge ad routes
	api.HandleFunc("/lounge-ads", loungeAdHandler.GetAll).Methods("GET", "OPTIONS")
	protected.HandleFunc("/lounge-ads", loungeAdHandler.Create).Methods("POST", "OPTIONS")
	protected.HandleFunc("/lounge-ads/{id}", loungeAdHandler.Update).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/lounge-ads/{id}", loungeAdHandler.Delete).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/lounge-ads/lounge/{loungeId}", loungeAdHandler.GetForLounge).Methods("GET", "OPTIONS")
	api.HandleFunc("/lounge-ads/slots/{loungeId}", loungeAdHandler.GetSlotSummary).Methods("GET", "OPTIONS")

	// Dashboard route
	api.HandleFunc("/dashboard", dashboardHandler.GetDashboardData).Methods("GET", "OPTIONS")

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}).Methods("GET", "OPTIONS")

	router.PathPrefix("/media/").Handler(http.StripPrefix("/media/", http.FileServer(http.Dir(cfg.MediaDir))))

	addr := fmt.Sprintf(":%s", cfg.ServerPort)
	log.Printf("Server starting on port %s", cfg.ServerPort)
	log.Fatal(http.ListenAndServe(addr, router))
}
