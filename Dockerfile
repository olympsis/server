# --- Local models context ---
# The go.mod `replace github.com/olympsis/models => ../models` needs the local
# models module available at build time. The module builds at /app, so the
# relative `../models` resolves to /models inside the image.
#   - Dev:  compose.dev.yaml overrides this named context with the real dir
#           (`additional_contexts: { models: ../models }`), so /models is populated.
#   - Prod: the replace is removed from go.mod, so this empty `scratch` fallback
#           is used and nothing is copied.
FROM scratch AS models

# --- Build stage ---
FROM golang:1.25-alpine AS builder
WORKDIR /app
# Place the (dev-only) local models module where the go.mod replace expects it.
COPY --from=models . /models
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /server

# --- Runtime stage ---
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /server /app/server

ARG STORAGE_URL

ENV PORT=80
ENV MODE=PRODUCTION

ENV MONGO_NAME=olympsis
ENV LOCAL_NAME=locales
ENV NOTIFICATIONS_NAME=notifications

ENV ANNOUNCEMENT_COLLECTION=announcements

ENV AUTHENTICATION_COLLECTION=auth
ENV USER_COLLECTION=users
ENV FRIEND_REQUEST_COLLECTION=friendRequests

ENV POSTS_COLLECTION=posts
ENV POST_COMMENTS_COLLECTION=postComments
ENV POST_REACTIONS_COLLECTION=postReactions

ENV CLUB_COLLECTION=clubs
ENV CLUB_INVITE_COLLECTION=clubInvites
ENV CLUB_MEMBERS_COLLECTION=clubMembers
ENV CLUB_INVITATION_COLLECTION=clubInvitations
ENV CLUB_APPLICATION_COLLECTION=clubApplications

ENV ORGANIZATION_COLLECTION=organizations
ENV ORGANIZATION_INVITATION_COLLECTION=organizationInvitations
ENV ORGANIZATION_APPLICATION_COLLECTION=organizationApplications
ENV ORGANIZATION_MEMBERS_COLLECTION=organizationMembers

ENV VENUES_COLLECTION=venues
ENV VENUE_UNITS_COLLECTION=venueUnits
ENV TRANSIT_LINES_COLLECTION=transitLines

ENV EVENTS_COLLECTION=events
ENV EVENT_LOGS_COLLECTION=eventLogs
ENV EVENT_VIEWS_COLLECTION=eventViews
ENV EVENT_TEAMS_COLLECTION=eventTeams
ENV EVENT_COMMENTS_COLLECTION=eventComments
ENV EVENT_INVITATIONS_COLLECTION=eventInvitations
ENV EVENT_PARTICIPANTS_COLLECTION=eventParticipants

ENV CLUB_TRANSACTIONS_COLLECTION=clubTransactions
ENV CLUB_FINANCIAL_ACCOUNTS_COLLECTION=clubFinancialAccounts

ENV EVENT_INVITATION_COLLECTION=eventInvitations
ENV CLUB_INVITATION_COLLECTION=clubInvitations
ENV ORG_INVITATION_COLLECTION=organizationInvitations

ENV BUG_REPORT_COLLECTION=bugReports
ENV POST_REPORT_COLLECTION=postReports
ENV VENUE_REPORT_COLLECTION=venueReports
ENV EVENT_REPORT_COLLECTION=eventReports
ENV MEMBER_REPORT_COLLECTION=memberReports

ENV LOCALE_NAME=locales
ENV COUNTRY_COLLECTION=countries
ENV ADMIN_AREA_COLLECTION=administrativeAreas
ENV SUB_ADMIN_AREA_COLLECTION=subAdministrativeAreas

ENV NOTIFICATIONS_NAME=notifications
ENV NOTIFICATIONS_COLLECTION=notes
ENV NOTIFICATION_LOGS_COLLECTION=logs
ENV USER_NOTIFICATIONS_COLLECTION=users
ENV NOTIFICATION_TOPICS_COLLECTION=topics

ENV TAGS_COLLECTION=tags
ENV SPORTS_COLLECTION=sports

EXPOSE 80

CMD ["/app/server"]
