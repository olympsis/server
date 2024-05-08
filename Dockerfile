FROM golang:1.20-bullseye
WORKDIR /app
COPY ./ ./
RUN go build -o /docker
RUN go mod download

ENV PORT=80
ENV MODE=PRODUCTION

ENV KEYID=JN25FUC9X2
ENV TEAMID=5A6H49Q85D

ENV DB_USR=admin
ENV DB_PASS=vM9pPgfHeZDxgBDv

ENV DB_NAME=olympsis
ENV DB_ADDR=database-0.i4q7nvi.mongodb.net
ENV NOTIF_DB_ADDR=database-2.pdjjqal.mongodb.net
ENV NOTIF_DB_NAME=notifications
ENV NOTIF_COL=topics

ENV AUTH_COL=auth
ENV USER_COL=users
ENV CLUB_COL=clubs
ENV ORG_COL=organizations
ENV EVENT_COL=events
ENV FIELD_COL=fields
ENV POST_COL=posts
ENV CINVITE_COL=clubInvites
ENV COMMENTS_COL=comments
ENV FREQUEST_COL=friendRequests
ENV CAPPICATIONS_COL=clubApplications
ENV OAPPICATIONS_COL=organizationApplications

ENV EVENT_INVITATIONS_COL=eventInvitations
ENV CLUB_INVITATIONS_COL=clubInvitations
ENV ORG_INVITATIONS_COL=organizationInvitations

ENV BUG_REPORT_COL=bugReports
ENV POST_REPORT_COL=postReports
ENV FIELD_REPORT_COL=fieldReports
ENV EVENT_REPORT_COL=eventReports
ENV MEMBER_REPORT_COL=memberReports

ENV KEY=SZkp78avQkxGyjRakxb5Ob08zqjguNRA

EXPOSE 80

CMD ["/docker"]