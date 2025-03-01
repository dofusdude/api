package database

import "time"

type BonusType struct {
	ID        int64      `db:"id"`
	NameID    string     `db:"name_id"`
	NameEn    string     `db:"name_en"`
	NameFr    string     `db:"name_fr"`
	NameEs    string     `db:"name_es"`
	NameDe    string     `db:"name_de"`
	NamePt    string     `db:"name_pt"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

type Bonus struct {
	ID            int64      `db:"id"`
	BonusTypeID   int64      `db:"bonus_type_id"`
	DescriptionEn string     `db:"description_en"`
	DescriptionFr string     `db:"description_fr"`
	DescriptionEs string     `db:"description_es"`
	DescriptionDe string     `db:"description_de"`
	DescriptionPt string     `db:"description_pt"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
	DeletedAt     *time.Time `db:"deleted_at"`
}

type Tribute struct {
	ID             int64      `db:"id"`
	ItemNameEn     string     `db:"item_name_en"`
	ItemNameFr     string     `db:"item_name_fr"`
	ItemNameEs     string     `db:"item_name_es"`
	ItemNameDe     string     `db:"item_name_de"`
	ItemNamePt     string     `db:"item_name_pt"`
	ItemAnkamaID   int64      `db:"item_ankama_id"`
	ItemCategoryId int        `db:"item_category_id"`
	ItemDoduapiUri string     `db:"item_doduapi_uri"`
	Quantity       int        `db:"quantity"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
	DeletedAt      *time.Time `db:"deleted_at"`
}

type Almanax struct {
	ID          int64      `db:"id"`
	BonusID     int64      `db:"bonus_id"`
	TributeID   int64      `db:"tribute_id"`
	Date        string     `db:"date"`
	RewardKamas int64      `db:"reward_kamas"`
	XpRatio     float64    `db:"xp_ratio"`
	OptimalLvl  int        `db:"optimal_lvl"`
	Duration    float64    `db:"duration"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"`
}

type MappedAlmanax struct {
	Almanax   Almanax
	Bonus     Bonus
	BonusType BonusType
	Tribute   Tribute
}
