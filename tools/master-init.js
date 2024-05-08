db = db.getSiblingDB("olympsis");

db.createCollection("auth");
db.createCollection("bugReports");

db.createCollection("clubApplications");
db.createCollection("clubInvitations");
db.createCollection("clubs");

db.createCollection("eventInvitations");
db.createCollection("eventReports");
db.createCollection("events");

db.createCollection("fieldReports");
db.createCollection("fields");

db.createCollection("groupReports");
db.createCollection("memberReports");

db.createCollection("organizationApplications");
db.createCollection("organizationInvitations");
db.createCollection("organizations");

db.createCollection("postReports");
db.createCollection("posts");

db.createCollection("users");

db.fields.createIndex({ location: "2dsphere" });

db.fields.insert({
    "_id": {
        "$oid": "639aa6f00710e5e04e653677"
    },
    "owner": {
        "name": "Orem City",
        "type": "public"
    },
    "name": "Palisade Park",
    "sports": [
        "soccer",
        "pickleball"
    ],
    "images": [
        "field-images/22aa23f7-5fb5-4c2e-b4fb-3a2943d492fc.jpg"
    ],
    "location": {
        "type": "Point",
        "coordinates": [
        -111.664209,
        40.314095
        ]
    },
    "city": "Orem",
    "state": "UT",
    "country": "United States",
    "description": "grass field"
});

db.fields.insert({
    "_id": {
      "$oid": "639b89cc56d46c7ad459c336"
    },
    "owner": {
      "name": "Brigham Young University",
      "type": "private"
    },
    "name": "Richard Building Fields",
    "sports": [
      "soccer",
      "pickleball"
    ],
    "images": [
      "field-images/a6c468f9-f259-49b4-8a94-a355139573b8.jpg"
    ],
    "location": {
      "type": "Point",
      "coordinates": [
        -111.655317,
        40.24948
      ]
    },
    "city": "Provo",
    "state": "UT",
    "country": "United States",
    "description": "The BYU Richard building fields are a collection of outdoor athletic fields located on the campus of Brigham Young University in Provo, Utah. The fields are primarily used by the university's intramural and club sports teams, as well as for physical education classes. The fields include multiple soccer, football, softball, and baseball fields. The Richard building fields are also used for recreational activities like pick-up games, and also host various events and tournaments throughout the year. "
});

db.fields.insert({
    "_id": {
      "$oid": "644d3a19b05e1c00c86f66f2"
    },
    "name": "11th Ave Park",
    "owner": {
      "name": "Salt Lake City",
      "type": "public"
    },
    "description": "The 11th ave park is a newly built park in the avenues. It is a multi-purposed park, featuring basketball, volleyball and tennis courts. It also features a walking trail and a drinking fountain.",
    "sports": [
      "basketball",
      "volleyball",
      "tennis"
    ],
    "images": [
      "field-images/71a7614f-6dd8-4f25-8b29-7fd3c0a3fe0e.jpg"
    ],
    "location": {
      "type": "Point",
      "coordinates": [
        -111.862922,
        40.783499
      ]
    },
    "city": "Salt Lake City",
    "state": "UT",
    "country": "United States of America"
});