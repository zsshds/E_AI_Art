package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	uri := flag.String("uri", "mongodb://localhost:27017/imagegen", "MongoDB URI")
	dbName := flag.String("db", "imagegen", "Database name")
	username := flag.String("username", "", "Admin username (required)")
	password := flag.String("password", "", "Admin password (required)")
	flag.Parse()

	if *username == "" || *password == "" {
		fmt.Fprintf(os.Stderr, "Usage: createadmin -username <name> -password <pass> [-uri <mongo_uri>] [-db <db_name>]\n")
		os.Exit(1)
	}

	if len(*username) < 3 {
		log.Fatal("username must be at least 3 characters")
	}
	if len(*password) < 6 {
		log.Fatal("password must be at least 6 characters")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(*uri))
	if err != nil {
		log.Fatalf("connect to mongodb: %v", err)
	}
	defer client.Disconnect(context.Background())

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("ping mongodb: %v", err)
	}

	col := client.Database(*dbName).Collection("users")

	// Check if user already exists
	var existing struct {
		Username string `bson:"username"`
	}
	err = col.FindOne(ctx, bson.M{"username": *username}).Decode(&existing)
	if err == nil {
		// User exists — update password and ensure admin role
		hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("hash password: %v", err)
		}
		_, err = col.UpdateOne(ctx,
			bson.M{"username": *username},
			bson.M{"$set": bson.M{
				"password_hash": string(hash),
				"role":          "admin",
			}},
		)
		if err != nil {
			log.Fatalf("update user: %v", err)
		}
		log.Printf("Admin user '%s' updated (password reset, role set to admin)", *username)
		return
	}

	// Create new admin user
	hash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	_, err = col.InsertOne(ctx, bson.M{
		"username":      *username,
		"password_hash": string(hash),
		"role":          "admin",
		"created_at":    time.Now(),
	})
	if err != nil {
		log.Fatalf("create user: %v", err)
	}

	log.Printf("Admin user '%s' created successfully", *username)
}
