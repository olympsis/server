#!/bin/bash

# Setting environment variables
export PORT=80
export MODE=PRODUCTION

export DB_USR=server
export DB_PASS=RvmeTaUvkGs7Vc8e
export DB_NAME=olympsis
export DB_ADDR=production.md0v8.mongodb.net

export AUTH_COL=auth
export USER_COL=users
export CLUB_COL=clubs
export ORG_COL=organizations
export EVENT_COL=events
export FIELD_COL=fields
export POST_COL=posts
export CLUB_INVITE_COL=clubInvites
export COMMENTS_COL=comments
export FRIEND_REQUEST_COL=friendRequests
export CLUB_APPLICATIONS_COL=clubApplications
export ORG_APPLICATIONS_COL=organizationApplications

export EVENT_INVITATIONS_COL=eventInvitations
export CLUB_INVITATIONS_COL=clubInvitations
export ORG_INVITATIONS_COL=organizationInvitations

export BUG_REPORT_COL=bugReports
export POST_REPORT_COL=postReports
export FIELD_REPORT_COL=fieldReports
export EVENT_REPORT_COL=eventReports
export MEMBER_REPORT_COL=memberReports

export NOTIFY=true
export NOTIF_URL=https://notifications-p3dy744wqq-uc.a.run.app

echo "Environment variables have been set."