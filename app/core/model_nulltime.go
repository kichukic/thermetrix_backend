package core

import (
	"time"

	"database/sql/driver"
	"encoding/json"
	"reflect"
	"strconv"
)

type NullTime struct {
	Time  time.Time
	Valid bool // Valid is true if Time is not NULL
}

func Now() NullTime {
	return NullTime{Time: time.Now(), Valid: true}
}

func (u *NullTime) FromString(s string) {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.999999999Z0700", s)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05", s)
			if err != nil {
				t, err = time.Parse("2006-01-02T15:04:05Z07:00", s)
				if err != nil {
					t, err = time.Parse("2006-01-02 15:04:05", s)
					if err != nil {
						t, err = time.Parse("2006-01-02 15:04:05.999", s)
						if err != nil {
							t, err = time.Parse("2006-01-02", s)
							if err != nil {
								t, err = time.Parse("2006-01-02T15:04:05Z", s)
								if err != nil {
									t, err = time.Parse("02/1/2006", s)
									if err != nil {
										t, err = time.Parse("02/01/2006", s)
										if err != nil {
											t, err = time.Parse("2/01/2006", s)
											if err != nil {
												t, err = time.Parse("02/01/2006", s)
												if err != nil {
													t, err = time.Parse("02/01/06", s)
													if err != nil {
														t, err = time.Parse("02/1/06", s)
														if err != nil {
															t, err = time.Parse("2/1/06", s)
															if err != nil {
																t, err = time.Parse("2/01/06", s)
																i, err := strconv.Atoi(s)
																t = time.Unix(int64(i), 0)
																if err != nil {
																	u.Time = t
																	u.Valid = false
																	return
																}
															}
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}

		}
	}
	u.Time = t
	u.Valid = true

}

func (u *NullTime) UnmarshalJSON(data []byte) error {

	s := string(data)

	// Get rid of the quotes "" around the value.
	// A second option would be to include them
	// in the date format string instead, like so below:
	//   time.Parse(`"`+time.RFC3339Nano+`"`, s)
	s = s[1 : len(s)-1]

	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, err = time.Parse("2006-01-02T15:04:05.999999999Z0700", s)
		if err != nil {
			t, err = time.Parse("2006-01-02T15:04:05", s)
			if err != nil {
				t, err = time.Parse("2006-01-02T15:04:05Z07:00", s)
				if err != nil {
					t, err = time.Parse("2006-01-02 15:04:05", s)
					if err != nil {
						t, err = time.Parse("2006-01-02 15:04:05.999", s)
						if err != nil {
							t, err = time.Parse("2006-01-02", s)
							if err != nil {
								t, err = time.Parse("2006-01-02T15:04:05Z", s)
								if err != nil {
									i, err := strconv.Atoi(s)
									t = time.Unix(int64(i), 0)
									if err != nil {
										u.Time = t
										u.Valid = false
										return nil
									}
								}
							}
						}
					}
				}
			}

		}
	}
	u.Time = t
	u.Valid = true
	return nil

}

/*
func (u *NullTime) MarshalJSON() ([]byte, error) {
	if u == nil {
		return json.Marshal("")
	}

	if( u.Valid ) {
		//		log.Println("TIME: ", u.Time.String())
		if(u.Time.String() == "0001-01-01 00:00:00 +0000 UTC"){
			return json.Marshal("")
		}
		return json.Marshal(u.Time)
	} else {
		return json.Marshal("")
	}


}
*/

func (u NullTime) MarshalJSON() ([]byte, error) {

	if u.Valid {
		//		log.Println("TIME: ", u.Time.String())
		if u.Time.String() == "0001-01-01 00:00:00 +0000 UTC" {
			return json.Marshal("")
		}
		return json.Marshal(u.Time)
	} else {
		return json.Marshal("")
	}

}

// Scan implements the Scanner interface.
func (nt *NullTime) Scan(value interface{}) error {
	//log.Println(value.(time.Time))
	nt.Time, nt.Valid = value.(time.Time)
	if !nt.Valid && value != nil {
		if reflect.TypeOf(value).String() == "[]uint8" {
			t, err := time.Parse("2006-01-02", string(value.([]uint8)))
			if err == nil {
				nt.Time = t
				nt.Valid = true
			}
		} else if reflect.TypeOf(value).String() == "*NullTime" {
			t := value.(*NullTime)
			nt.Time = t.Time
			nt.Valid = t.Valid
		}
	}

	return nil
}

// Value implements the driver Valuer interface.
func (nt NullTime) Value() (driver.Value, error) {
	if !nt.Valid {
		return nil, nil
	}
	return nt.Time, nil
}
