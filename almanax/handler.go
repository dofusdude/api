package almanax

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/dofusdude/doduapi/config"
	"github.com/dofusdude/doduapi/database"
	e "github.com/dofusdude/doduapi/errmsg"
	"github.com/dofusdude/doduapi/utils"
	mapping "github.com/dofusdude/dodumap"
	"github.com/hashicorp/go-memdb"
	"github.com/meilisearch/meilisearch-go"
)

var bonusDescriptionTemplateRe = regexp.MustCompile(`{{([^,]+),([0-9]+)::([^{]+)}}`)

func GetAlmanaxSingle(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	date := r.Context().Value("date").(time.Time)
	level := r.URL.Query().Get("level")

	var levelInt *int
	if level != "" {
		levelParse, err := strconv.Atoi(level)
		if err != nil {
			e.WriteInvalidQueryResponse(w, "Invalid level value.")
			return
		}

		if levelParse < 1 || levelParse > 200 {
			e.WriteInvalidQueryResponse(w, "Level value out of bounds.")
			return
		}

		levelInt = &levelParse
	}

	almDb := database.NewDatabaseRepository(context.Background(), config.DbDir)
	defer almDb.Deinit()

	dateStr := date.Format("2006-01-02")
	mappedAlmanax, err := almDb.GetAlmanaxByDateRange(dateStr, dateStr)
	if err != nil {
		e.WriteServerErrorResponse(w, "Database error while getting Almanax.")
		return
	}

	if len(mappedAlmanax) == 0 {
		e.WriteNotFoundResponse(w, "No Almanax found.")
		return
	}

	if len(mappedAlmanax) > 1 {
		e.WriteServerErrorResponse(w, "Multiple Almanax found for the same date.")
		return
	}

	itemDb := database.Db.Txn(false)
	defer itemDb.Abort()

	response, err := renderAlmanaxResponse(&mappedAlmanax[0], lang, levelInt, itemDb)
	if err != nil {
		e.WriteServerErrorResponse(w, "Could not render Almanax response. "+err.Error())
		return
	}

	utils.RequestsTotal.Inc()
	utils.RequestsAlmanaxSingle.Inc()

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		e.WriteServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func experienceReward(playerLevel, optimalLevel int, xpRatio, duration float64) int {
	if playerLevel == -1 {
		playerLevel = 200
	}

	if playerLevel > optimalLevel {
		rewardLevel := int(math.Min(float64(playerLevel), float64(optimalLevel)*1.5))
		fixeOptimalLevelExperienceReward := float64(optimalLevel) * math.Pow(100.0+2.0*float64(optimalLevel), 2.0) / 20.0 * duration * xpRatio
		fixeLevelExperienceReward := float64(rewardLevel) * math.Pow(100.0+2.0*float64(rewardLevel), 2.0) / 20.0 * duration * xpRatio

		reducedOptimalExperienceReward := (1.0 - 0.7) * fixeOptimalLevelExperienceReward
		reducedExperienceReward := 0.7 * fixeLevelExperienceReward
		return int(math.Floor(reducedExperienceReward + reducedOptimalExperienceReward))
	}
	return int(math.Floor(float64(playerLevel) * math.Pow(100.0+2.0*float64(playerLevel), 2.0) / 20.0 * duration * xpRatio))
}

func renderAlmanaxResponse(m *database.MappedAlmanax, lang string, level *int, txn *memdb.Txn) (AlmanaxResponse, error) {
	var response AlmanaxResponse
	response.Date = m.Almanax.Date
	response.Bonus.BonusType.Id = m.BonusType.NameID
	response.Tribute.Quantity = m.Tribute.Quantity
	response.RewardKamas = int(m.Almanax.RewardKamas)
	response.Tribute.Item.AnkamaId = m.Tribute.ItemAnkamaID
	response.Tribute.Item.Subtype = utils.CategoryIdApiMapping(m.Tribute.ItemCategoryId)

	categoryDbType := utils.CategoryIdMapping(m.Tribute.ItemCategoryId)

	raw, err := txn.First(fmt.Sprintf("%s-%s", utils.CurrentRedBlueVersionStr(database.Version.MemDb), categoryDbType), "id", response.Tribute.Item.AnkamaId)
	if err != nil {
		return response, err
	}

	if raw == nil {
		return response, fmt.Errorf("Item %d not found in %s database.", response.Tribute.Item.AnkamaId, categoryDbType)
	}

	item := raw.(*mapping.MappedMultilangItemUnity)
	response.Tribute.Item.ImageUrls = RenderImageUrls(utils.ImageUrls(item.IconId, "item", config.ItemImgResolutions, config.ApiScheme, config.MajorVersion, config.ApiHostName, config.IsBeta))

	switch lang {
	case "en":
		response.Bonus.Description = m.Bonus.DescriptionEn
		response.Bonus.BonusType.Name = m.BonusType.NameEn
		response.Tribute.Item.Name = m.Tribute.ItemNameEn
	case "fr":
		response.Bonus.Description = m.Bonus.DescriptionFr
		response.Bonus.BonusType.Name = m.BonusType.NameFr
		response.Tribute.Item.Name = m.Tribute.ItemNameFr
	case "de":
		response.Bonus.Description = m.Bonus.DescriptionDe
		response.Bonus.BonusType.Name = m.BonusType.NameDe
		response.Tribute.Item.Name = m.Tribute.ItemNameDe
	case "es":
		response.Bonus.Description = m.Bonus.DescriptionEs
		response.Bonus.BonusType.Name = m.BonusType.NameEs
		response.Tribute.Item.Name = m.Tribute.ItemNameEs
	case "pt":
		response.Bonus.Description = m.Bonus.DescriptionPt
		response.Bonus.BonusType.Name = m.BonusType.NamePt
		response.Tribute.Item.Name = m.Tribute.ItemNamePt
	}

	// replace templated links inside the bonus description; TODO replace this later with a meta link to the linked item or monster
	response.Bonus.Description = bonusDescriptionTemplateRe.ReplaceAllStringFunc(response.Bonus.Description, func(match string) string {
		group := bonusDescriptionTemplateRe.FindStringSubmatch(match)
		if len(group) < 4 {
			return match // return original if for some reason we do not have enough captures
		}
		return group[3] // only return the last capture group, which is the localized name
	})

	if level != nil {
		response.RewardXp = new(int)
		*response.RewardXp = experienceReward(*level, m.Almanax.OptimalLvl, m.Almanax.XpRatio, m.Almanax.Duration)
	}

	return response, nil
}

func GetAlmanaxRange(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	from := r.URL.Query().Get("range[from]")
	to := r.URL.Query().Get("range[to]")
	size := r.URL.Query().Get("range[size]")
	var sizeNum int
	bonusType := r.URL.Query().Get("filter[bonus_type]")
	timezone := r.URL.Query().Get("timezone")

	level := r.URL.Query().Get("level")
	var levelInt *int
	if level != "" {
		levelParse, err := strconv.Atoi(level)
		if err != nil {
			e.WriteInvalidQueryResponse(w, "Invalid level value.")
			return
		}

		if levelParse < 1 || levelParse > 200 {
			e.WriteInvalidQueryResponse(w, "Level value out of bounds.")
			return
		}

		levelInt = &levelParse
	}

	var err error

	if size == "" {
		sizeNum = -1
	} else {
		sizeNum, err = strconv.Atoi(size)
		if err != nil {
			e.WriteInvalidQueryResponse(w, "Invalid size value.")
			return
		}
	}

	if timezone == "" {
		timezone = "Europe/Paris"
	}

	givenFromDate := from != ""
	var fromDate time.Time
	var fromDateParsed time.Time
	if givenFromDate {
		fromDateParsed, err = time.Parse("2006-01-02", from)
		if err != nil {
			e.WriteInvalidQueryResponse(w, "Invalid from-date format.")
			return
		}
	}

	givenToDate := to != ""
	var toDate time.Time
	var toDateParsed time.Time
	if givenToDate {
		toDateParsed, err = time.Parse("2006-01-02", to)
		if err != nil {
			e.WriteInvalidQueryResponse(w, "Invalid to-date format.")
			return
		}
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		e.WriteInvalidQueryResponse(w, "Invalid timezone.")
		return
	}
	fromDate = time.Now().In(loc)
	toDate = fromDate.AddDate(0, 0, config.AlmanaxDefaultLookAhead)

	givenRangeSize := size != "" && sizeNum > 0
	if givenRangeSize && givenFromDate && givenToDate {
		e.WriteInvalidQueryResponse(w, "Cannot use range[size] with range[from] and range[to].")
		return
	}

	if givenRangeSize && !givenFromDate && !givenToDate {
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			e.WriteInvalidQueryResponse(w, "Invalid timezone.")
			return
		}
		fromDate = time.Now().In(loc)
		toDate = fromDate.AddDate(0, 0, sizeNum)
	} else {
		if givenFromDate && givenToDate {
			fromDate = fromDateParsed
			toDate = toDateParsed
		}

		if givenFromDate && !givenToDate {
			fromDate = fromDateParsed
			toDate = fromDate.AddDate(0, 0, config.AlmanaxDefaultLookAhead)
		}

		if !givenFromDate && givenToDate {
			toDate = toDateParsed
		}
	}

	if givenRangeSize && givenFromDate {
		toDate = fromDate.AddDate(0, 0, sizeNum)
	}

	if givenRangeSize && givenToDate {
		toDate = toDateParsed
		fromDate = toDate.AddDate(0, 0, -sizeNum)
	}

	if fromDate.After(toDate) {
		e.WriteInvalidQueryResponse(w, "From-date is after to-date.")
		return
	}

	if toDate.Sub(fromDate).Hours() > float64(config.AlmanaxMaxLookAhead)*24 {
		e.WriteInvalidQueryResponse(w, "Date range is too large.")
		return
	}

	almDb := database.NewDatabaseRepository(context.Background(), config.DbDir)
	defer almDb.Deinit()

	if bonusType != "" {
		bonusTypes, err := almDb.GetBonusTypes()
		if err != nil {
			e.WriteServerErrorResponse(w, "Could not get bonus types. "+err.Error())
			return
		}

		found := false
		for _, b := range bonusTypes {
			if b.NameID == bonusType {
				found = true
				break
			}
		}

		if !found {
			e.WriteInvalidQueryResponse(w, "Invalid bonus type.")
			return
		}
	}

	itemDb := database.Db.Txn(false)
	defer itemDb.Abort()

	fromDateStr := fromDate.Format("2006-01-02")
	toDateStr := toDate.Format("2006-01-02")

	res := make([]AlmanaxResponse, 0)
	var mappedAlmanax []database.MappedAlmanax
	if bonusType != "" {
		mappedAlmanax, err = almDb.GetAlmanaxByDateRangeAndNameID(fromDateStr, toDateStr, bonusType)
		if err != nil {
			e.WriteServerErrorResponse(w, "Database error while getting Almanax with bonus type. "+err.Error())
			return
		}
	} else {
		mappedAlmanax, err = almDb.GetAlmanaxByDateRange(fromDateStr, toDateStr)
		if err != nil {
			e.WriteServerErrorResponse(w, "Database error while getting Almanax. "+err.Error())
			return
		}
	}

	if len(mappedAlmanax) == 0 {
		e.WriteNotFoundResponse(w, "No Almanax found.")
		return
	}

	for _, m := range mappedAlmanax {
		response, err := renderAlmanaxResponse(&m, lang, levelInt, itemDb)
		if err != nil {
			e.WriteServerErrorResponse(w, "Could not render Almanax response. "+err.Error())
			return
		}
		res = append(res, response)
	}

	utils.RequestsTotal.Inc()
	utils.RequestsAlmanaxRange.Inc()

	utils.WriteCacheHeader(&w)
	encodeErr := json.NewEncoder(w).Encode(res)
	if encodeErr != nil {
		e.WriteServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func bonusListingsToBonusIdTranslated(bonuses []database.BonusType, lang string) []AlmanaxBonusListing {
	bonusesTranslated := make([]AlmanaxBonusListing, 0, len(bonuses))
	for _, bonus := range bonuses {
		var bonusTranslated AlmanaxBonusListing
		bonusTranslated.Id = bonus.NameID
		switch lang {
		case "en":
			bonusTranslated.Name = bonus.NameEn
		case "fr":
			bonusTranslated.Name = bonus.NameFr
		case "de":
			bonusTranslated.Name = bonus.NameDe
		case "es":
			bonusTranslated.Name = bonus.NameEs
		case "pt":
			bonusTranslated.Name = bonus.NamePt
		}
		bonusesTranslated = append(bonusesTranslated, bonusTranslated)
	}

	return bonusesTranslated
}

func ListBonuses(w http.ResponseWriter, r *http.Request) {
	lang := r.Context().Value("lang").(string)
	db := database.NewDatabaseRepository(context.Background(), config.DbDir)

	bonuses, err := db.GetBonusTypes()
	if err != nil {
		e.WriteServerErrorResponse(w, "Could not get bonus types: "+err.Error())
		return
	}

	if len(bonuses) == 0 {
		e.WriteNotFoundResponse(w, "No bonuses found.")
		return
	}

	bonusesTranslated := bonusListingsToBonusIdTranslated(bonuses, lang)

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(bonusesTranslated)
	if err != nil {
		e.WriteServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}

func getLimitInBoundary(limitStr string) (int64, error) {
	if limitStr == "" {
		limitStr = "8"
	}
	var limit int
	var err error
	if limit, err = strconv.Atoi(limitStr); err != nil {
		return 0, fmt.Errorf("invalid limit value")
	}
	if limit > 100 {
		return 0, fmt.Errorf("limit value is too high")
	}

	return int64(limit), nil
}

func SetJsonHeader(w *http.ResponseWriter) {
	(*w).Header().Set("Content-Type", "application/json")
}

func SearchBonuses(w http.ResponseWriter, r *http.Request) {
	client := meilisearch.New(config.MeiliHost, meilisearch.WithAPIKey(config.MeiliKey))
	defer client.Close()

	query := r.URL.Query().Get("query")
	if query == "" {
		e.WriteInvalidQueryResponse(w, "Query parameter is required.")
		return
	}

	lang := r.Context().Value("lang").(string)

	var searchLimit int64
	var err error
	if searchLimit, err = getLimitInBoundary(r.URL.Query().Get("limit")); err != nil {
		e.WriteInvalidQueryResponse(w, "Invalid limit value: "+err.Error())
		return
	}

	index := client.Index(fmt.Sprintf("alm-bonuses-%s", lang))

	request := &meilisearch.SearchRequest{
		Limit: searchLimit,
	}

	var searchResp *meilisearch.SearchResponse
	if searchResp, err = index.Search(query, request); err != nil {
		e.WriteServerErrorResponse(w, "Could not search: "+err.Error())
		return
	}

	//requestsTotal.Inc()
	//requestsSearchTotal.Inc()

	if searchResp.EstimatedTotalHits == 0 {
		e.WriteNotFoundResponse(w, "No results found.")
		return
	}

	var results []AlmanaxBonusListing
	for _, hit := range searchResp.Hits {
		almBonusJson := hit.(map[string]interface{})
		almBonus := AlmanaxBonusListing{
			Id:   almBonusJson["slug"].(string),
			Name: almBonusJson["name"].(string),
		}
		results = append(results, almBonus)
	}

	utils.WriteCacheHeader(&w)
	err = json.NewEncoder(w).Encode(results)
	if err != nil {
		e.WriteServerErrorResponse(w, "Could not encode JSON: "+err.Error())
		return
	}
}
