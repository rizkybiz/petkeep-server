package api

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql" //mysql driver
	"golang.org/x/crypto/bcrypt"
)

//connectDB connects to a MYSQL database
func connectDB(host, port, user, password, database string) (*sql.DB, error) {
	connString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, database)
	log.Println(connString)
	conn, err := sql.Open("mysql", connString)
	if err != nil {
		return nil, err
	}
	err = conn.Ping()
	if err != nil {
		return nil, err
	}
	log.Println("Successfully connected to MySQL")
	return conn, nil
}

//disconnectDB disconnects from a MYSQL database
func disconnectDB(db *sql.DB) error {
	err := db.Close()
	if err != nil {
		return err
	}
	return nil
}

//MigrateDB runs the database migrations to build the schema
func migrateDB(db *sql.DB) error {
	usersTableMigration := `CREATE TABLE IF NOT EXISTS users (
		id int AUTO_INCREMENT,
		email varchar(60) NOT NULL UNIQUE,
		password varchar(60) NOT NULL,
		created_at timestamp NOT NULL,
		updated_at timestamp NOT NULL,
		last_login timestamp,
		PRIMARY KEY(id)) ENGINE=INNODB`
	petsTableMigration := `CREATE TABLE IF NOT EXISTS pets (
			id int AUTO_INCREMENT,
			user_id int NOT NULL,
			name varchar(60) NOT NULL,
			type varchar(30),
			breed varchar(40),
			birthday date,
			PRIMARY KEY (id),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE) ENGINE=INNODB`
	_, err := db.Exec(usersTableMigration)
	if err != nil {
		return err
	}
	_, err = db.Exec(petsTableMigration)
	if err != nil {
		return err
	}
	return nil
}

//dbLogin handles checking if a user exists
func (s *server) dbLogin(email, password string) (int64, error) {

	//Check if user exists
	var id int64
	var storedPass string
	row := s.db.QueryRow("SELECT id,password FROM users WHERE email = ?", email)
	err := row.Scan(&id, &storedPass)
	if err != nil {
		return 0, err
	}

	// Compare passwords
	err = bcrypt.CompareHashAndPassword([]byte(storedPass), []byte(password))
	if err != nil {
		return 0, err
	}

	//Update lastLogin
	ts := time.Now()
	_, err = s.db.Exec("UPDATE users SET last_login = ? WHERE email = ?", ts, email)

	//return ID
	return id, nil
}

//dbUsersCreate handles the validation and creation of a new user
func (s *server) dbUsersCreate(u user) (int64, error) {

	//Insert user into users table
	result, err := s.db.Exec("INSERT INTO users(email, password, created_at, updated_at) VALUES(?,?,?,?)", u.Email, u.Password, u.CreatedAt.Format("2006-01-02 15:04:05"), u.UpdatedAt.Format("2006-01-02 15:04:05"))
	if err != nil {
		return 0, err
	}

	//Get inserted user's id and return
	idInt, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return idInt, nil
}

//dbUsersCreate handles the validation and creation of a new user
func (s *server) dbUsersGetOne(id int64) (u user, e error) {

	//Get user from db
	row := s.db.QueryRow("SELECT id, email, created_at, updated_at, last_login FROM users WHERE id = ?", id)
	err := row.Scan(&u.ID, &u.Email, &u.CreatedAt, &u.UpdatedAt, &u.LastLogin)
	if err != nil {
		e = err
		return u, err
	}
	return u, nil
}

//dbPetsGetAll returns all pets owned by a user by ID
func (s *server) dbPetsGetAll(id int64) ([]pet, error) {

	// Get pets from db
	rows, err := s.db.Query("SELECT * FROM pets WHERE user_id = ?", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pets []pet

	// Iterate through results and append to a slice
	for rows.Next() {
		var pet pet
		err := rows.Scan(&pet.ID, &pet.UserID, &pet.Name, &pet.Type, &pet.Breed, &pet.Birthday)
		if err != nil {
			return nil, err
		}
		pets = append(pets, pet)
	}
	return pets, nil
}
