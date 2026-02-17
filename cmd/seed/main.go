package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

func main() {
	connStr := "postgres://postgres:password@localhost:5432/volumetric_db?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("DB ping failed:", err)
	}
	fmt.Println("Connected to database successfully")

	// ── 1. Super Admin (upsert + get real user_id) ──
	email := "admin@example.com"
	var userID uuid.UUID

	err = db.QueryRow(`
		INSERT INTO users (user_id, full_name, email, password_hash, is_active, is_email_verified)
		VALUES ($1, $2, $3, $4, true, true)
		ON CONFLICT (email) DO UPDATE 
		    SET full_name = EXCLUDED.full_name,
		        password_hash = EXCLUDED.password_hash
		RETURNING user_id
	`, uuid.New(), "Super Admin", email, "temp123456").
		Scan(&userID)

	if err != nil {
		log.Fatal("Failed to upsert super admin:", err)
	}

	// Assign role (safe even if already exists)
	_, err = db.Exec(`
		INSERT INTO user_role (user_id, role_id)
		VALUES ($1, 1)  -- 1 = super_admin
		ON CONFLICT DO NOTHING
	`, userID)
	if err != nil {
		log.Fatal("Failed to assign role:", err)
	}

	// ── 2. Seed one Vehicle (matches your real columns) ──
	_, err = db.Exec(`
		INSERT INTO vehicles (
			vehicle_number, 
			type, 
			capacity, 
			driver_name, 
			site_id, 
			created_at
		) VALUES (
			'PB10AB1234',
			'Tipper Truck',
			20.00,
			'Ram Singh',
			NULL,           -- site_id can be NULL
			now()
		) ON CONFLICT DO NOTHING
	`)
	if err != nil {
		log.Fatal("Failed to seed vehicle:", err)
	}

	// ── 3. Seed one Material Type ──
	_, err = db.Exec(`
		INSERT INTO material_types (name, density_kg_per_m3, created_at)
		VALUES ('Sand', 1600.0000, now())
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		log.Fatal("Failed to seed material type:", err)
	}

	fmt.Println("──────────────────────────────────────────────")
	fmt.Println("Seeding completed successfully!")
	fmt.Println("Super Admin:")
	fmt.Println("  Email:    admin@example.com")
	fmt.Println("  Password: temp123456 (plain for dev)")
	fmt.Println("  user_id: ", userID.String())
	fmt.Println("")
	fmt.Println("Test Vehicle: vehicle_number = 'PB10AB1234' (id likely 1)")
	fmt.Println("Test Material Type: name = 'Sand' (id likely 1)")
	fmt.Println("──────────────────────────────────────────────")
	fmt.Println("Now test protected /scans with:")
	fmt.Println("  vehicle_id: 1")
	fmt.Println("  material_type_id: 1")
}