const appDb = db.getSiblingDB("imagegen");
const appUser = process.env.MONGO_APP_USER;
const appPass = process.env.MONGO_APP_PASS;

if (!appUser || !appPass) {
  throw new Error("MONGO_APP_USER and MONGO_APP_PASS must be set");
}

if (!appDb.getUser(appUser)) {
  appDb.createUser({
    user: appUser,
    pwd: appPass,
    roles: [{ role: "readWrite", db: "imagegen" }],
  });
}

function ensureCollection(name) {
  if (!appDb.getCollectionNames().includes(name)) {
    appDb.createCollection(name);
  }
}

ensureCollection("style_profiles");
appDb.style_profiles.createIndex({ created_by: 1 });
appDb.style_profiles.createIndex({ is_locked: 1 });
appDb.style_profiles.createIndex({ created_at: -1 });

ensureCollection("tasks");
appDb.tasks.createIndex({ style_profile_id: 1 });
appDb.tasks.createIndex({ status: 1 });
appDb.tasks.createIndex({ created_by: 1 });
appDb.tasks.createIndex({ created_at: -1 });
appDb.tasks.createIndex({ status: 1, created_at: 1 });

ensureCollection("users");
appDb.users.createIndex({ username: 1 }, { unique: true });
