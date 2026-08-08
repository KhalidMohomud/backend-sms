package model

import "time"

type AcademicYear struct {
	ID        uint64    `gorm:"column:y_no;primaryKey;uniqueIndex:uq_academic_years_id_school" json:"id"`
	SchoolID  uint64    `gorm:"column:sch_no;not null;uniqueIndex:uq_academic_year_school_name;uniqueIndex:uq_academic_years_id_school" json:"school_id"`
	YearName  string    `gorm:"column:year_name;size:20;not null;uniqueIndex:uq_academic_year_school_name" json:"year_name"`
	StartsOn  time.Time `gorm:"column:started;type:date;not null" json:"starts_on"`
	EndsOn    time.Time `gorm:"column:ended;type:date;not null;check:ck_academic_year_dates,ended > started" json:"ends_on"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
	School    School    `gorm:"foreignKey:SchoolID;references:ID" json:"-"`
}

func (AcademicYear) TableName() string { return "academic_years" }
