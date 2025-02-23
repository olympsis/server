package utils

import "os"

// Reads from OS environment variables to create server config object
func GetServerConfig() ServerConfig {
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}

	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "DEVELOPMENT"
	}

	http := os.Getenv("HTTP")
	if http == "" {
		http = "UNSECURE"
	}

	firebase := os.Getenv("FIREBASE_FILE_PATH")
	if firebase == "" {
		panic("firebase file path missing in config")
	}

	keyPath := os.Getenv("KEY_FILE_PATH")
	certPath := os.Getenv("CERT_FILE_PATH")
	if http == "SECURE" && (keyPath == "" || certPath == "") {
		panic("secure server requires a key and certificate file path env variable")
	}

	mapkit := os.Getenv("MAPKIT_TOKEN")

	notif := os.Getenv("NOTIF_URL")
	storage := os.Getenv("STORAGE_URL")

	return ServerConfig{
		Port:             port,
		Mode:             mode,
		Http:             http,
		FirebaseFilePath: firebase,

		MapKitToken: mapkit,

		NotifServiceURL:   notif,
		StorageServiceURL: storage,

		KeyFilePath:  keyPath,
		CertFilePath: certPath,
	}
}

// Reads from OS environment variables to create a database config object
func GetDatabaseConfig() DatabaseConfig {
	name := os.Getenv("DB_NAME")
	if name == "" {
		panic("database name required in config")
	}

	addr := os.Getenv("DB_ADDR")
	if addr == "" {
		panic("database address required in config")
	}

	user := os.Getenv("DB_USER")
	if user == "" {
		panic("database user required in config")
	}

	pass := os.Getenv("DB_PASS")
	if pass == "" {
		panic("database password required in config")
	}

	localeDB := os.Getenv("COUNTRY_COL")
	if localeDB == "" {
		panic("locale database name required in config")
	}

	return DatabaseConfig{
		Name:     name,
		Address:  addr,
		User:     user,
		Password: pass,

		LocaleName: localeDB,
	}
}

// Reads from OS environment variables to create a collections config object
func GetCollectionsConfig() CollectionsConfig {
	// USER COLLECTIONS
	authCollection := os.Getenv("AUTH_COL")
	if authCollection == "" {
		panic("auth collection name required in config")
	}
	userCollection := os.Getenv("USER_COL")
	if userCollection == "" {
		panic("user collection name required in config")
	}

	// CLUB COLLECTIONS
	clubCollection := os.Getenv("CLUB_COL")
	if clubCollection == "" {
		panic("club collection name required in config")
	}
	clubInvitationCollection := os.Getenv("CLUB_INVITATION_COL")
	if clubInvitationCollection == "" {
		panic("club invitation collection name required in config")
	}
	clubApplicationCollection := os.Getenv("CLUB_APPLICATION_COL")
	if clubApplicationCollection == "" {
		panic("club application collection name required in config")
	}

	// ORGANIZATION COLLECTIONS
	orgCollection := os.Getenv("ORG_COL")
	if orgCollection == "" {
		panic("organization collection name is required in config")
	}
	orgInvitationCollection := os.Getenv("ORG_INVITATION_COL")
	if orgInvitationCollection == "" {
		panic("organization invitation name collection required in config")
	}
	orgApplicationCollection := os.Getenv("ORG_APPLICATION_COL")
	if orgApplicationCollection == "" {
		panic("organization application name collection required in config")
	}

	// EVENT COLLECTIONS
	eventCollection := os.Getenv("EVENT_COL")
	if eventCollection == "" {
		panic("event collection name required in config")
	}
	eventActivityCollection := os.Getenv("EVENT_ACTIVITY_COL")
	// if eventActivityCollection == "" {
	// 	panic("event activity collection name required in config")
	// }
	eventInvitationCollection := os.Getenv("EVENT_INVITATION_COL")
	if eventInvitationCollection == "" {
		panic("event invitation collection name required in config")
	}

	// VENUE COLLECTIONS
	venueCollection := os.Getenv("VENUE_COL")
	if venueCollection == "" {
		panic("venue collection name required in config")
	}
	venueRequestCollection := os.Getenv("VENUE_REQUEST_COL")
	// if venueRequestCollection == "" {
	// 	panic("venue request collection name required in config")
	// }

	// POST COLLECTIONS
	postCollection := os.Getenv("POST_COL")
	if postCollection == "" {
		panic("post collection name required in config")
	}
	commentCollection := os.Getenv("COMMENT_COL")
	if commentCollection == "" {
		panic("comment collection name required in config")
	}

	// REPORT COLLECTIONS
	bugReportCollection := os.Getenv("BUG_REPORT_COL")
	if bugReportCollection == "" {
		panic("bug report collection name required in config")
	}
	venueReportCollection := os.Getenv("VENUE_REPORT_COL")
	// if venueReportCollection == "" {
	// 	panic("venue report collection name required in config")
	// }
	eventReportCollection := os.Getenv("EVENT_REPORT_COL")
	if eventReportCollection == "" {
		panic("event report collection name required in config")
	}
	memberReportCollection := os.Getenv("MEMBER_REPORT_COL")
	if memberReportCollection == "" {
		panic("member report collection name required in config")
	}

	// LOG COLLECTIONS
	authLog := os.Getenv("AUTH_LOG_COL")
	// if authLog == "" {
	// 	panic("auth log collection name required in config")
	// }
	eventLog := os.Getenv("EVENT_LOG_COL")
	// if eventLog == "" {
	// 	panic("event log collection name required in config")
	// }
	venueLog := os.Getenv("VENUE_LOG_COL")
	// if venueLog == "" {
	// 	panic("venue log collection name required in config")
	// }
	postLog := os.Getenv("POST_LOG_COL")
	// if postLog == "" {
	// 	panic("post log collection name required in config")
	// }
	commentLog := os.Getenv("COMMENT_LOG_COL")
	// if commentLog == "" {
	// 	panic("comment log collection name required in config")
	// }
	orgLog := os.Getenv("ORG_LOG_COL")
	// if orgLog == "" {
	// 	panic("org log collection name required in config")
	// }
	clubLog := os.Getenv("CLUB_LOG_COL")
	// if clubLog == "" {
	// 	panic("club log collection name required in config")
	// }
	clubApplicationLog := os.Getenv("CLUB_APPLICATION_LOG_COL")
	// if clubApplicationLog == "" {
	// 	panic("club application log collection name required in config")
	// }

	// LOCALE COLLECTIONS
	countriesCollection := os.Getenv("COUNTRY_COL")
	if countriesCollection == "" {
		panic("countries collection name required in config")
	}
	adminAreaCollection := os.Getenv("ADMIN_AREA_COL")
	if adminAreaCollection == "" {
		panic("admin area collection name required in config")
	}
	subAdminAreaCollection := os.Getenv("SUB_ADMIN_AREA_COL")
	if subAdminAreaCollection == "" {
		panic("sub admin area collection name required in config")
	}

	return CollectionsConfig{
		AuthCollection: authCollection,
		UserCollection: userCollection,

		ClubCollection:            clubCollection,
		ClubInvitationCollection:  clubInvitationCollection,
		ClubApplicationCollection: clubApplicationCollection,

		OrgCollection:            orgCollection,
		OrgInvitationCollection:  orgInvitationCollection,
		OrgApplicationCollection: orgApplicationCollection,

		EventCollection:           eventCollection,
		EventActivityCollection:   eventActivityCollection,
		EventInvitationCollection: eventInvitationCollection,

		VenueCollection:        venueCollection,
		VenueRequestCollection: venueRequestCollection,

		PostCollection:    postCollection,
		CommentCollection: commentCollection,

		BugReportCollection:    bugReportCollection,
		VenueReportCollection:  venueReportCollection,
		EventReportCollection:  eventReportCollection,
		MemberReportCollection: memberReportCollection,

		AuthLogs:            authLog,
		EventLogs:           eventLog,
		VenueLogs:           venueLog,
		PostLogs:            postLog,
		CommentLogs:         commentLog,
		OrgLogs:             orgLog,
		ClubLogs:            clubLog,
		ClubApplicationLogs: clubApplicationLog,

		CountriesCollection:     countriesCollection,
		AdminAreasCollection:    adminAreaCollection,
		SubAdminAreasCollection: subAdminAreaCollection,
	}
}
