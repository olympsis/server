FROM golang:1.22-bullseye
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
ENV CLUB_INVITE_COL=clubInvites
ENV COMMENTS_COL=comments
ENV FRIEND_REQUEST_COL=friendRequests
ENV CLUB_APPLICATIONS_COL=clubApplications
ENV ORG_APPLICATIONS_COL=organizationApplications

ENV EVENT_INVITATIONS_COL=eventInvitations
ENV CLUB_INVITATIONS_COL=clubInvitations
ENV ORG_INVITATIONS_COL=organizationInvitations

ENV BUG_REPORT_COL=bugReports
ENV POST_REPORT_COL=postReports
ENV FIELD_REPORT_COL=fieldReports
ENV EVENT_REPORT_COL=eventReports
ENV MEMBER_REPORT_COL=memberReports

ENV KEY=SZkp78avQkxGyjRakxb5Ob08zqjguNRA
ENV SECRET=4aE8ENjmjnEAivkm9kDzXkq+oEsH5EWGbnrUdHBK72g/gJe8RF6B+G90wJp9o+LlEwMU/hqglKLE/nrzf9qUmw==

ENV NOTIFY=true
ENV NOTIF_URL=https://notifications-p3dy744wqq-uc.a.run.app

EXPOSE 80

CMD ["/docker"]