package migrate

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	"github.com/stretchr/testify/assert"
)

var (
	errEmpty      = errors.New("[empty error]")
	database      *pg.DB
	defaultLogger Logger
)

const (
	zero int64 = 0
)

type testLogger struct{}

func (*testLogger) Infof(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func init() {
	defaultLogger = new(testLogger)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	database = pg.Connect(&pg.Options{
		Addr:     os.Getenv("TEST_DATABASE_ADDR"),
		User:     os.Getenv("TEST_DATABASE_USER"),
		Password: os.Getenv("TEST_DATABASE_PASS"),
		Database: os.Getenv("TEST_DATABASE_DB"),
		PoolSize: 2,
	})

	createTables(database)

	database.Exec("TRUNCATE ?", getTableName())
}

type mockDB struct {
	*pg.DB
	*pg.Tx
}

func (m *mockDB) RunInTransaction(fn func(*pg.Tx) error) error {
	if m.DB == nil {
		return errEmpty
	}
	return m.DB.RunInTransaction(fn)
}
func (m *mockDB) Exec(query interface{}, params ...interface{}) (orm.Result, error) {
	if m.Tx == nil {
		return nil, errEmpty
	}
	return m.Tx.Exec(query, params...)
}
func (m *mockDB) ExecOne(query interface{}, params ...interface{}) (orm.Result, error) {
	if m.Tx == nil {
		return nil, errEmpty
	}
	return m.Tx.ExecOne(query, params...)
}
func (m *mockDB) Query(model, query interface{}, params ...interface{}) (orm.Result, error) {
	if m.Tx == nil {
		return nil, errEmpty
	}
	return m.Tx.Query(model, query, params...)
}
func (m *mockDB) QueryOne(model, query interface{}, params ...interface{}) (orm.Result, error) {
	if m.Tx == nil {
		return nil, errEmpty
	}
	return m.Tx.QueryOne(model, query, params...)
}

func TestMigrate_List(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		database.RunInTransaction(func(tx *pg.Tx) error {
			m, errNew := New(Options{
				DB:     &mockDB{DB: database, Tx: tx},
				Path:   "./fixtures/good",
				Logger: defaultLogger,
			})

			if !assert.NoError(t, errNew) {
				return errNew
			}

			if errUp := m.Up(0); !assert.NoError(t, errUp) {
				return errUp
			}

			items, errList := m.List()
			if !assert.NoError(t, errList) {
				return errList
			}

			assert.True(t, len(m.(*migrate).Migrations) == len(items))

			return errEmpty
		})
	})

	t.Run("Bad", func(t *testing.T) {
		database.RunInTransaction(func(tx *pg.Tx) error {
			m, errNew := New(Options{
				DB:     &mockDB{DB: database, Tx: tx},
				Path:   "./fixtures/good",
				Logger: defaultLogger,
			})

			if !assert.NoError(t, errNew) {
				return errNew
			}

			if errUp := m.Up(0); !assert.NoError(t, errUp) {
				return errUp
			}

			m.(*migrate).DB = &mockDB{DB: database, Tx: nil}

			_, errList := m.List()
			if !assert.Error(t, errList) {
				return errors.New("must be error")
			}

			return errEmpty
		})
	})
}

func TestMigrate_Plan(t *testing.T) {
	t.Run("Good case #1", func(t *testing.T) {
		database.RunInTransaction(func(tx *pg.Tx) error {
			m, errNew := New(Options{
				DB:     &mockDB{DB: database, Tx: tx},
				Path:   "./fixtures/good",
				Logger: defaultLogger,
			})

			if !assert.NoError(t, errNew) {
				return errNew
			}

			if errUp := m.Up(0); !assert.NoError(t, errUp) {
				return errUp
			}

			items, errList := m.Plan()
			if !assert.NoError(t, errList) {
				return errList
			}

			assert.True(t, len(items) == 0)

			return errEmpty
		})
	})

	t.Run("Good case #2", func(t *testing.T) {
		database.RunInTransaction(func(tx *pg.Tx) error {
			m, errNew := New(Options{
				DB:     &mockDB{DB: database, Tx: tx},
				Path:   "./fixtures/good",
				Logger: defaultLogger,
			})

			if !assert.NoError(t, errNew) {
				return errNew
			}

			if errDown := m.Down(2); !assert.NoError(t, errDown) {
				return errDown
			}

			items, errList := m.Plan()
			if !assert.NoError(t, errList) {
				return errList
			}

			assert.True(t, len(m.(*migrate).Migrations) == len(items), spew.Sdump(items, m.(*migrate).Migrations))

			return errEmpty
		})
	})

	t.Run("Bad", func(t *testing.T) {
		database.RunInTransaction(func(tx *pg.Tx) error {
			m, errNew := New(Options{
				DB:     &mockDB{DB: database, Tx: tx},
				Path:   "./fixtures/good",
				Logger: defaultLogger,
			})

			if !assert.NoError(t, errNew) {
				return errNew
			}

			if errUp := m.Up(0); !assert.NoError(t, errUp) {
				return errUp
			}

			m.(*migrate).DB = &mockDB{DB: database, Tx: nil}

			_, errList := m.Plan()
			if !assert.Error(t, errList) {
				return errors.New("must be error")
			}

			return errEmpty
		})
	})
}

func TestUpDown(t *testing.T) {
	t.Run("Good", func(t *testing.T) {
		database.RunInTransaction(func(tx *pg.Tx) error {
			m, errNew := New(Options{
				DB: &mockDB{
					DB: database,
					Tx: tx,
				},
				Path:   "./fixtures/good",
				Logger: defaultLogger,
			})

			if !assert.NoError(t, errNew) {
				return errNew
			}

			if errUp := m.Up(0); !assert.NoError(t, errUp) {
				return errUp
			}

			if errDown := m.Down(0); !assert.NoError(t, errDown) {
				return errDown
			}

			return errEmpty
		})

	})

	t.Run("Bad", func(t *testing.T) {
		database.RunInTransaction(func(tx *pg.Tx) error {

			m, errNew := New(Options{
				DB: &mockDB{
					DB: database,
					Tx: tx,
				},
				Path:   "./fixtures/bad",
				Logger: defaultLogger,
			})

			if !assert.NoError(t, errNew) {
				return errNew
			}

			if errUp := m.Up(0); !assert.Error(t, errUp) {
				return errUp
			}

			mig, ok := m.(*migrate)
			if !assert.True(t, ok) {
				return errors.New("not migrate")
			}

			for _, item := range mig.Migrations {
				if errVer := addVersion(tx, item.Version, item.RealName()); !assert.NoError(t, errVer) {
					return errVer
				}
			}

			if errDown := m.Down(0); !assert.Error(t, errDown) {
				return errDown
			}

			for _, item := range mig.Migrations {
				if errVer := remVersion(tx, item.Version, item.RealName()); !assert.NoError(t, errVer) {
					return errVer
				}
			}

			return errEmpty
		})
	})

	t.Run("Bad Up / Down / Version", func(t *testing.T) {
		database.RunInTransaction(func(tx *pg.Tx) error {

			var (
				mEmpty  = new(mockDB)
				mNormal = &mockDB{
					DB: database,
					Tx: tx,
				}
			)

			m, errNew := New(Options{
				DB:     mNormal,
				Path:   "./fixtures/bad",
				Logger: defaultLogger,
			})

			if !assert.NoError(t, errNew) {
				return errNew
			}

			t.Run("UP", func(t *testing.T) {
				if errUp := m.Up(-1); !assert.Error(t, errUp) {
					t.Fatal("must be error", errUp)
				}
			})

			t.Run("UP cannot fetch version", func(t *testing.T) {
				m.(*migrate).DB = mEmpty
				if errUp := m.Up(0); !assert.Error(t, errUp) {
					t.Fatal("must be error", errUp)
				}
			})

			m.(*migrate).DB = mNormal

			t.Run("DOWN", func(t *testing.T) {
				if errDown := m.Down(-1); !assert.Error(t, errDown) {
					t.Fatal("must be error", errDown)
				}
			})

			t.Run("Down cannot fetch version", func(t *testing.T) {
				m.(*migrate).DB = mEmpty
				if errDown := m.Down(0); !assert.Error(t, errDown) {
					t.Fatal("must be error", errDown)
				}
			})

			m.(*migrate).DB = mNormal

			t.Run("Version", func(t *testing.T) {
				m.(*migrate).DB = &mockDB{}
				if _, errVer := m.Version(); !assert.Error(t, errVer) {
					t.Fatal("must be error", errVer)
				}
			})

			return errEmpty
		})
	})
}

func TestNew(t *testing.T) {
	var err error

	t.Run("Good case", func(t *testing.T) {
		if err = database.RunInTransaction(func(tx *pg.Tx) error {
			tx.Exec(`TRUNCATE ?`, getTableName())
			m, errNew := New(Options{
				DB:     &mockDB{DB: database, Tx: tx},
				Path:   "./fixtures/good",
				Logger: defaultLogger,
			})

			if !assert.NoError(t, errNew) {
				return fmt.Errorf("new err: %v", errNew)
			}

			if !assert.NotNil(t, m) {
				return errors.New("empty migrator")
			}

			if ver, errVer := m.Version(); !assert.NoError(t, errVer) {
				return fmt.Errorf("version err: %v", errVer)
			} else if !assert.Equal(t, zero, ver) {
				return fmt.Errorf("wrong migration version: %d != %d", zero, ver)
			}

			var i int64

			for i = 1; i <= 10; i++ {
				if _, errVer := tx.Exec(
					sqlNewVersion,
					getTableName(),
					i,
					strconv.FormatInt(i, 10)+"_test",
				); errVer != nil {
					return fmt.Errorf("version err: %v", errVer)
				}

				if ver, errVer := m.Version(); !assert.NoError(t, errVer) {
					return fmt.Errorf("version err: %v", errVer)
				} else if !assert.Equal(t, i, ver) {
					return fmt.Errorf("wrong migration version: %d != %d", i, ver)
				}
			}

			// to reject inserts
			return errEmpty
		}); err != nil && err != errEmpty {
			t.Fatal(err)
		}
	})

	t.Run("Bad case", func(t *testing.T) {
		_, err = New(Options{
			DB: nil,
		})
		assert.Error(t, err)
	})
}

func TestMigrate_Version(t *testing.T) {
	database.RunInTransaction(func(tx *pg.Tx) error {
		tx.Exec(`TRUNCATE ?`, getTableName())
		m := migrate{
			Options: Options{
				DB:     &mockDB{DB: database, Tx: tx},
				Path:   "./fixtures/good",
				Logger: defaultLogger,
			},
		}

		// Good case:
		if ver, errVer := m.Version(); !assert.NoError(t, errVer) {
			return errVer
		} else if !assert.Equal(t, zero, ver) {
			return fmt.Errorf("wrong migration version: %d != %d", zero, ver)
		}

		// Bad case #1:
		m.DB = &mockDB{Tx: nil}

		if _, errVer := m.Version(); !assert.Error(t, errVer) {
			return errVer
		}

		// Bad case #2:
		m.DB = &mockDB{Tx: nil}

		if _, errVer := m.Version(); !assert.Error(t, errVer) {
			return errVer
		}

		// to reject inserts
		return errEmpty
	})
}
