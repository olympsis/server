package utils

import (
	"errors"
	"olympsis-server/utils/secrets"
	"os"

	"github.com/sideshow/apns2"
	"github.com/sideshow/apns2/token"
)

// Create APNS2 Client
func CreateApns2Client(keyID string, teamID string, fileName string) (*apns2.Client, error) {
	key, err := token.AuthKeyFromFile(fileName)
	if err != nil {
		return nil, errors.New("failed to read key from file. Error: " + err.Error())
	}

	token := token.Token{
		AuthKey: key,
		KeyID:   keyID,
		TeamID:  teamID,
	}

	mode := os.Getenv("MODE")
	switch mode {
	case "PRODUCTION":
		return apns2.NewTokenClient(&token).Production(), nil
	default:
		return apns2.NewTokenClient(&token).Development(), nil
	}
}

// Reads from OS environment variables to create server config object
func GetServerConfig(manager *secrets.Manager) ServerConfig {
	config := ServerConfig{}

	// Port
	port := os.Getenv("PORT")
	if port == "" {
		port = "80"
	}
	config.Port = port

	// Server mode
	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "DEVELOPMENT"
	}
	config.Mode = mode

	// Secure/Unsecure modes
	http := os.Getenv("HTTP")
	if http == "" {
		http = "UNSECURE"
	}
	config.Http = http

	// Apple Key ID
	config.AppleKeyID = manager.GetRequired("APPLE_KEY_ID")

	// Apple Team ID
	config.AppleTeamID = manager.GetRequired("APPLE_TEAM_ID")

	// p8 Key file path
	apnsFilePath := os.Getenv("APNS_FILE_PATH")
	if apnsFilePath == "" {
		if config.Mode == "PRODUCTION" {
			panic("No APNS key file path provided!")
		} else {
			apnsFilePath = "./files/AuthKey_5MP3VW78BZ.p8"
		}
	}
	config.APNSFileURl = apnsFilePath

	// Firebase config
	firebase := os.Getenv("FIREBASE_FILE_PATH")
	if firebase == "" {
		if config.Mode == "PRODUCTION" {
			panic("firebase file path missing in config")
		} else {
			firebase = "./files/firebase-credentials.json"
		}
	}
	config.FirebaseFilePath = firebase

	// Server SSL key & cert
	if config.Http == "SECURE" {
		keyPath := os.Getenv("KEY_FILE_PATH")
		certPath := os.Getenv("CERT_FILE_PATH")
		if keyPath == "" || certPath == "" {
			panic("secure server requires a key and certificate file path env variable")
		}

		config.KeyFilePath = keyPath
		config.CertFilePath = certPath
	}

	// GCP credentials file for Storage & Vision APIs
	gcpCreds := os.Getenv("STORAGE_FILE_PATH")
	if gcpCreds == "" {
		panic("GCS credentials file path required in config")
	}
	config.GCPCredentialsFilePath = gcpCreds

	// Set up MapKit token (static, for production /v1/token endpoint)
	config.MapKitToken = manager.GetRequired("MAPKIT_TOKEN")

	// Set up MapKit JWT generation config (for snapshot API and dev mode)
	mapkitFilePath := os.Getenv("MAPKIT_FILE_PATH")
	mapkitKeyID := os.Getenv("MAPKIT_KEY_ID")
	config.MapKitConfig = MapKitConfig{
		KeyFilePath: mapkitFilePath,
		KeyID:       mapkitKeyID,
		TeamID:      config.AppleTeamID,
	}

	return config
}

// Reads from OS environment variables to create a database config object
func GetDatabaseConfig(manager *secrets.Manager) DatabaseConfig {
	name := manager.GetRequired("MONGO_NAME")
	if name == "" {
		panic("database name required in config")
	}

	addr := manager.GetRequired("MONGO_ADDRESS")
	if addr == "" {
		panic("database address required in config")
	}

	user := manager.GetRequired("MONGO_USERNAME")
	if user == "" {
		panic("database user required in config")
	}

	pass := manager.GetRequired("MONGO_PASSWORD")
	if pass == "" {
		panic("database password required in config")
	}

	localeDB := manager.GetRequired("LOCAL_NAME")
	if localeDB == "" {
		panic("locale database name required in config")
	}

	noteDB := manager.GetRequired("NOTIFICATIONS_NAME")
	if noteDB == "" {
		panic("notifications database name required in config")
	}

	return DatabaseConfig{
		Name:     name,
		Address:  addr,
		User:     user,
		Password: pass,

		LocaleName:       localeDB,
		NotificationName: noteDB,
	}
}

func GetRedisConfig(manager *secrets.Manager) RedisConfig {
	address := manager.GetRequired("REDIS_ADDRESS")

	return RedisConfig{
		Address: address,
	}
}

// Reads from OS environment variables to create a collections config object
func GetCollectionsConfig() CollectionsConfig {

	announcementCollection := os.Getenv("ANNOUNCEMENT_COLLECTION")
	if announcementCollection == "" {
		panic("announcement collection required in config")
	}

	// USER COLLECTIONS
	authCollection := os.Getenv("AUTHENTICATION_COLLECTION")
	if authCollection == "" {
		panic("auth collection name required in config")
	}
	userCollection := os.Getenv("USER_COLLECTION")
	if userCollection == "" {
		panic("user collection name required in config")
	}

	// CLUB COLLECTIONS
	clubCollection := os.Getenv("CLUB_COLLECTION")
	if clubCollection == "" {
		panic("club collection name required in config")
	}
	clubMembersCollection := os.Getenv("CLUB_MEMBERS_COLLECTION")
	if clubMembersCollection == "" {
		panic("club members collection name required in config")
	}
	clubInvitationCollection := os.Getenv("CLUB_INVITATION_COLLECTION")
	if clubInvitationCollection == "" {
		panic("club invitation collection name required in config")
	}
	clubApplicationCollection := os.Getenv("CLUB_APPLICATION_COLLECTION")
	if clubApplicationCollection == "" {
		panic("club application collection name required in config")
	}
	clubFinancialAccountsCollection := os.Getenv("CLUB_FINANCIAL_ACCOUNTS_COLLECTION")
	if clubFinancialAccountsCollection == "" {
		panic("club financial accounts collection name required in config")
	}
	clubTransactionsCollection := os.Getenv("CLUB_TRANSACTIONS_COLLECTION")
	if clubTransactionsCollection == "" {
		panic("club transactions collection name required in config")
	}

	// ORGANIZATION COLLECTIONS
	orgCollection := os.Getenv("ORGANIZATION_COLLECTION")
	if orgCollection == "" {
		panic("organization collection name is required in config")
	}
	orgInvitationCollection := os.Getenv("ORGANIZATION_INVITATION_COLLECTION")
	if orgInvitationCollection == "" {
		panic("organization invitation name collection required in config")
	}
	orgApplicationCollection := os.Getenv("ORGANIZATION_APPLICATION_COLLECTION")
	if orgApplicationCollection == "" {
		panic("organization application name collection required in config")
	}
	organizationMembersCollection := os.Getenv("ORGANIZATION_MEMBERS_COLLECTION")
	if organizationMembersCollection == "" {
		panic("organization members name collection required in config")
	}

	// EVENT COLLECTIONS
	eventsCollection := os.Getenv("EVENTS_COLLECTION")
	if eventsCollection == "" {
		panic("events collection name required in config")
	}
	eventLogsCollection := os.Getenv("EVENT_LOGS_COLLECTION")
	if eventLogsCollection == "" {
		panic("event logs collection name required in config")
	}
	eventViewsCollection := os.Getenv("EVENT_VIEWS_COLLECTION")
	if eventViewsCollection == "" {
		panic("event views collection name required in config")
	}
	eventTeamsCollection := os.Getenv("EVENT_TEAMS_COLLECTION")
	if eventTeamsCollection == "" {
		panic("event teams collection name required in config")
	}
	eventCommentsCollection := os.Getenv("EVENT_COMMENTS_COLLECTION")
	if eventCommentsCollection == "" {
		panic("event comments collection name required in config")
	}
	eventInvitationsCollection := os.Getenv("EVENT_INVITATIONS_COLLECTION")
	if eventInvitationsCollection == "" {
		panic("event invitations collection name required in config")
	}
	eventParticipantsCollection := os.Getenv("EVENT_PARTICIPANTS_COLLECTION")
	if eventParticipantsCollection == "" {
		panic("event participants collection name required in config")
	}

	// VENUE COLLECTIONS
	venuesCollection := os.Getenv("VENUES_COLLECTION")
	if venuesCollection == "" {
		panic("venue collection name required in config")
	}
	venueRequestCollection := os.Getenv("VENUE_REQUEST_COL")
	// if venueRequestCollection == "" {
	// 	panic("venue request collection name required in config")
	// }

	// POST COLLECTIONS
	postCollection := os.Getenv("POSTS_COLLECTION")
	if postCollection == "" {
		panic("posts collection name required in config")
	}
	postCommentsCollection := os.Getenv("POST_COMMENTS_COLLECTION")
	if postCommentsCollection == "" {
		panic("post comments collection name required in config")
	}

	postReactionsCollection := os.Getenv("POST_REACTIONS_COLLECTION")
	if postReactionsCollection == "" {
		panic("post reactions collection name required in config")
	}

	// REPORT COLLECTIONS
	bugReportCollection := os.Getenv("BUG_REPORT_COLLECTION")
	if bugReportCollection == "" {
		panic("bug report collection name required in config")
	}
	venueReportCollection := os.Getenv("VENUE_REPORT_COLLECTION")
	// if venueReportCollection == "" {
	// 	panic("venue report collection name required in config")
	// }
	eventReportCollection := os.Getenv("EVENT_REPORT_COLLECTION")
	if eventReportCollection == "" {
		panic("event report collection name required in config")
	}
	memberReportCollection := os.Getenv("MEMBER_REPORT_COLLECTION")
	if memberReportCollection == "" {
		panic("member report collection name required in config")
	}

	// LOG COLLECTIONS
	authLog := os.Getenv("AUTH_LOG_COLLECTION")
	// if authLog == "" {
	// 	panic("auth log collection name required in config")
	// }
	eventLog := os.Getenv("EVENT_LOG_COLLECTION")
	// if eventLog == "" {
	// 	panic("event log collection name required in config")
	// }
	venueLog := os.Getenv("VENUE_LOG_COLLECTION")
	// if venueLog == "" {
	// 	panic("venue log collection name required in config")
	// }
	postLog := os.Getenv("POST_LOG_COLLECTION")
	// if postLog == "" {
	// 	panic("post log collection name required in config")
	// }
	commentLog := os.Getenv("COMMENT_LOG_COLLECTION")
	// if commentLog == "" {
	// 	panic("comment log collection name required in config")
	// }
	orgLog := os.Getenv("ORG_LOG_COLLECTION")
	// if orgLog == "" {
	// 	panic("org log collection name required in config")
	// }
	clubLog := os.Getenv("CLUB_LOG_COLLECTION")
	// if clubLog == "" {
	// 	panic("club log collection name required in config")
	// }
	clubApplicationLog := os.Getenv("CLUB_APPLICATION_LOG_COLLECTION")
	// if clubApplicationLog == "" {
	// 	panic("club application log collection name required in config")
	// }

	// LOCALE COLLECTIONS
	countriesCollection := os.Getenv("COUNTRY_COLLECTION")
	if countriesCollection == "" {
		panic("countries collection name required in config")
	}
	adminAreaCollection := os.Getenv("ADMIN_AREA_COLLECTION")
	if adminAreaCollection == "" {
		panic("admin area collection name required in config")
	}
	subAdminAreaCollection := os.Getenv("SUB_ADMIN_AREA_COLLECTION")
	if subAdminAreaCollection == "" {
		panic("sub admin area collection name required in config")
	}

	tagsCollection := os.Getenv("TAGS_COLLECTION")
	if tagsCollection == "" {
		panic("tags collection name required in config")
	}

	sportsCollection := os.Getenv("SPORTS_COLLECTION")
	if sportsCollection == "" {
		panic("sports collection name required in config")
	}

	notifications := os.Getenv("NOTIFICATIONS_COLLECTION")
	if notifications == "" {
		panic("notifications collection name required in config")
	}

	notificationLogs := os.Getenv("NOTIFICATION_LOGS_COLLECTION")
	if notificationLogs == "" {
		panic("notificationLogs collection name required in config")
	}

	userNotifications := os.Getenv("USER_NOTIFICATIONS_COLLECTION")
	if userNotifications == "" {
		panic("userNotifications collection name required in config")
	}

	notificationTopics := os.Getenv("NOTIFICATION_TOPICS_COLLECTION")
	if notificationTopics == "" {
		panic("notificationTopics collection name required in config")
	}

	return CollectionsConfig{
		AnnouncementCollection: announcementCollection,

		AuthCollection: authCollection,
		UserCollection: userCollection,

		ClubCollection:                  clubCollection,
		ClubMembersCollection:           clubMembersCollection,
		ClubInvitationCollection:        clubInvitationCollection,
		ClubApplicationCollection:       clubApplicationCollection,
		ClubTransactionsCollection:      clubTransactionsCollection,
		ClubFinancialAccountsCollection: clubFinancialAccountsCollection,

		OrgCollection:                 orgCollection,
		OrgInvitationCollection:       orgInvitationCollection,
		OrgApplicationCollection:      orgApplicationCollection,
		OrganizationMembersCollection: organizationMembersCollection,

		EventsCollection:            eventsCollection,
		EventLogsCollection:         eventLogsCollection,
		EventViewsCollection:        eventViewsCollection,
		EventTeamsCollection:        eventTeamsCollection,
		EventCommentsCollection:     eventCommentsCollection,
		EventInvitationsCollection:  eventInvitationsCollection,
		EventParticipantsCollection: eventParticipantsCollection,

		VenuesCollection:       venuesCollection,
		VenueRequestCollection: venueRequestCollection,

		PostCollection:          postCollection,
		PostCommentsCollection:  postCommentsCollection,
		PostReactionsCollection: postReactionsCollection,

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

		TagsCollections:  tagsCollection,
		SportsCollection: sportsCollection,

		NotificationsCollection:      notifications,
		NotificationLogsCollection:   notificationLogs,
		UserNotificationsCollection:  userNotifications,
		NotificationTopicsCollection: notificationTopics,
	}
}
