// Converts event participant RSVP statuses from the legacy integer enum to the
// string enum.
//
//   0 -> "MAYBE"   1 -> "YES"   2 -> "WAITLIST"
//
// ("CAN'T" is newer than the integer form, so nothing maps to it here.)
//
// Why this is required and not merely nice-to-have: models.RSVPStatus can still
// DECODE a legacy integer, so reads keep working without this. But queries
// cannot — a filter on "WAITLIST" will not match a document still holding 2, so
// waitlist promotion and the participants aggregation silently skip un-migrated
// records. Run this before (or immediately alongside) deploying the server build
// that writes strings.
//
// Usage:
//   mongosh "<connection-uri>" tools/migrate-rsvp-status.js
//
// Idempotent: re-running is a no-op because the filters only match integers.
// Safe to run while the server is up — the server reads both forms, and every
// write it makes is already a string.

db = db.getSiblingDB("olympsis");

const COLLECTION = "eventParticipants";
const MAPPING = [
    { code: 0, status: "MAYBE" },
    { code: 1, status: "YES" },
    { code: 2, status: "WAITLIST" },
];

const collection = db.getCollection(COLLECTION);

// Report what we're about to touch, so a dry inspection is possible by
// commenting out the updateMany below.
const before = collection.countDocuments({ status: { $type: ["int", "long", "double"] } });
print(`[rsvp-migration] ${COLLECTION}: ${before} document(s) with a numeric status`);

let migrated = 0;
for (const { code, status } of MAPPING) {
    // $type guards against the (impossible, but cheap to exclude) case of a
    // string that compares equal to a number.
    const result = collection.updateMany(
        { status: { $eq: code, $type: ["int", "long", "double"] } },
        { $set: { status: status } }
    );
    migrated += result.modifiedCount;
    print(`[rsvp-migration]   ${code} -> "${status}": ${result.modifiedCount} updated`);
}

// Anything numeric still left over is an unmapped code — surface it loudly
// rather than leaving it to fail at decode time.
const leftovers = collection.countDocuments({ status: { $type: ["int", "long", "double"] } });
if (leftovers > 0) {
    print(`[rsvp-migration] WARNING: ${leftovers} document(s) still hold an unmapped numeric status:`);
    collection
        .aggregate([
            { $match: { status: { $type: ["int", "long", "double"] } } },
            { $group: { _id: "$status", count: { $sum: 1 } } },
        ])
        .forEach((row) => print(`[rsvp-migration]   status=${row._id}: ${row.count}`));
}

print(`[rsvp-migration] done — ${migrated} document(s) migrated, ${leftovers} left`);
