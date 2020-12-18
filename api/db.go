package api

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" //postgresql driver
	"golang.org/x/crypto/bcrypt"
)

//connectDB connects to a cockroach database
func (s *server) connectDB(host, port, user, certPath, database string, dbInsecure bool) error {
	var connString string
	if dbInsecure {
		connString = fmt.Sprintf("postgresql://%s@%s:%s/%s?ssl=true&sslmode=disable", user, host, port, database)
	} else {
		connString = fmt.Sprintf("postgresql://%s@%s:%s/%s?ssl=true&sslmode=require&sslrootcert=%s/ca.crt&sslkey=%s/client.%s.key&sslcert=%s/client.%s.crt", user, host, port, database, certPath, certPath, user, certPath, user)
	}
	s.logger.Debug().Msg(fmt.Sprintf("connection string: %s", connString))
	conn, err := sql.Open("postgres", connString)
	if err != nil {
		return err
	}
	err = conn.Ping()
	if err != nil {
		return err
	}
	s.logger.Info().Msg("Successfully connected to cockroachdb")
	s.db = conn
	return nil
}

//disconnectDB disconnects from a cockroach database
func (s *server) disconnectDB() error {
	err := s.db.Close()
	if err != nil {
		return err
	}
	return nil
}

//MigrateDB runs the database migrations to build the schema
func migrateDB(db *sql.DB) error {
	usersTableMigration := `CREATE TABLE IF NOT EXISTS users (
			id SERIAL NOT NULL,
			email STRING UNIQUE,
			password STRING,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ,
			last_login TIMESTAMPTZ,
			PRIMARY KEY (id))`
	petsTableMigration := `CREATE TABLE IF NOT EXISTS pets (
			id SERIAL NOT NULL,
			user_id int REFERENCES users (id) ON DELETE CASCADE,
			name STRING NOT NULL,
			type STRING,
			gender STRING,
			breed STRING,
			birthday DATE,
			created_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ,
			PRIMARY KEY (id))`
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
	row := s.db.QueryRow("SELECT id,password FROM users WHERE email = $1", email)
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
	_, err = s.db.Exec("UPDATE users SET last_login = $1 WHERE email = $2", ts, email)

	//return ID
	return id, nil
}

//dbUsersCreate handles the validation and creation of a new user
func (s *server) dbUsersCreate(u user) (int64, error) {

	//Insert user into users table
	var id int64
	err := s.db.QueryRow("INSERT INTO users(email, password, created_at, updated_at) VALUES($1,$2,$3,$4) RETURNING id", u.Email, u.Password, u.CreatedAt, u.UpdatedAt).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

//dbUsersCreate handles the validation and creation of a new user
func (s *server) dbUsersGetOne(id int64) (u user, e error) {

	//Get user from db
	row := s.db.QueryRow("SELECT id, email, created_at, updated_at, last_login FROM users WHERE id = $1", id)
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
	rows, err := s.db.Query("SELECT * FROM pets WHERE user_id = $1", id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pets []pet

	// Iterate through results and append to a slice
	for rows.Next() {
		var pet pet
		err := rows.Scan(&pet.ID, &pet.UserID, &pet.Name, &pet.Type, &pet.Breed, &pet.Birthday, &pet.CreatedAt, &pet.UpdatedAt, &pet.Gender)
		if err != nil {
			return nil, err
		}
		pets = append(pets, pet)
	}
	return pets, nil
}

//dbPetsGetOne returns a single pet by ID owned by a user by ID
func (s *server) dbPetsGetOne(userID, petID int64) (p pet, e error) {

	// Get pet from db
	row := s.db.QueryRow("SELECT * FROM pets WHERE user_id =$1 AND id = $2", userID, petID)
	err := row.Scan(&p.ID, &p.UserID, &p.Name, &p.Type, &p.Gender, &p.Breed, &p.Birthday)
	if err != nil {
		return p, err
	}
	return p, nil
}

//dbUsersCreate handles the validation and creation of a new user
func (s *server) dbPetsCreate(p pet, userID int64) (int64, error) {

	//Insert pet into pets table
	q, err := s.db.Prepare("INSERT INTO pets(user_id, name, type, gender, breed, birthday, created_at, updated_at) VALUES($1,$2,$3,$4,$5,$6,$7, $8) RETURNING id")
	if err != nil {
		return 0, err
	}
	defer q.Close()

	var id int64
	err = q.QueryRow(userID, p.Name, p.Type, p.Gender, p.Breed, p.Birthday, p.CreatedAt, p.UpdatedAt).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

//dbPetsUpdate handles the updating of pets
func (s *server) dbPetsUpdate(p pet, userID int64) error {

	//Update Pet
	q, err := s.db.Prepare("UPDATE pets SET name = $1, type = $2, gender = $3, breed = $4, birthday = $5, updated_at = $6 id = $7, created_at = $8, user_id = $9 WHERE id = $10")
	if err != nil {
		return err
	}
	defer q.Close()

	_, err = q.Exec(p.Name, p.Type, p.Gender, p.Breed, p.Birthday, p.UpdatedAt, p.ID, p.CreatedAt, p.UserID, p.ID)
	if err != nil {
		return err
	}
	return nil
}

//dbPetsDelete handles the deletion of pets
func (s *server) dbPetsDelete(petID, userID int64) (int64, error) {

	//Delete pet
	q, err := s.db.Prepare("DELETE FROM pets WHERE id = $1 and user_id = $2")
	if err != nil {
		return 0, err
	}
	defer q.Close()

	res, err := q.Exec(petID, userID)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}
