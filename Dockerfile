FROM golang:1.20-bullseye
WORKDIR /app
COPY ./ ./
RUN go build -o /docker
RUN go mod download
ENV PORT=80
ENV DB_ADDR=host.docker.internal
ENV DB_USR=service
ENV DB_PASS=qN1PHHgo6L942AvpTgGQ
ENV DB_NAME=olympsis
ENV AUTH_COL=auth
ENV USER_COL=users
ENV CULB_COL=clubs
ENV EVENT_COL=events
ENV FIELD_COL=fields
ENV POST_COL=posts
ENV CINVITE_COL=clubInvites
ENV COMMENTS_COL=comments
ENV FREQUEST_COL=friendRequests
ENV CAPPICATIONS_COL=clubApplications
ENV KEY=SZkp78avQkxGyjRakxb5Ob08zqjguNRA

EXPOSE 80

CMD ["/docker"]