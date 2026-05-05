// 创建应用账号（权限隔离，不使用 root）
db.getSiblingDB("imagegen").createUser({
  user: process.env.MONGO_APP_USER || "imagegen_app",
  pwd: process.env.MONGO_APP_PASS || "changeme",
  roles: [{ role: "readWrite", db: "imagegen" }]
});

const db = db.getSiblingDB("imagegen");

// 创建集合 + 索引
db.createCollection("style_profiles");
db.style_profiles.createIndex({ "created_by": 1 });
db.style_profiles.createIndex({ "is_locked": 1 });
db.style_profiles.createIndex({ "created_at": -1 });

db.createCollection("tasks");
db.tasks.createIndex({ "style_profile_id": 1 });
db.tasks.createIndex({ "status": 1 });
db.tasks.createIndex({ "created_by": 1 });
db.tasks.createIndex({ "created_at": -1 });
// 用于 Worker 查询 pending 任务的复合索引
db.tasks.createIndex({ "status": 1, "created_at": 1 });

db.createCollection("users");
db.users.createIndex({ "username": 1 }, { unique: true });
