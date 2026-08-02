package persistence

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"smart-recruit-proto/recruitment/pb"
	"smart-recruit-recruitment-service/internal/domain/model"
	profilepkg "smart-recruit-recruitment-service/internal/domain/profile"
)

type candidateEducationRecord struct {
	ID          int64 `gorm:"primaryKey"`
	UserID      int64
	School      string
	Degree      sql.NullString
	Major       sql.NullString
	StartDate   *time.Time
	EndDate     *time.Time
	Description sql.NullString
	SortOrder   int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (candidateEducationRecord) TableName() string { return "candidate_educations" }

type candidateExperienceRecord struct {
	ID          int64 `gorm:"primaryKey"`
	UserID      int64
	Company     string
	Title       sql.NullString
	Location    sql.NullString
	StartDate   *time.Time
	EndDate     *time.Time
	IsCurrent   int32
	Description sql.NullString
	SortOrder   int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (candidateExperienceRecord) TableName() string { return "candidate_experiences" }

type resumeProfileRecord struct {
	ID                   int64 `gorm:"primaryKey"`
	ResumeID             int64
	UserID               int64
	ParseRunID           int64
	IsCurrent            int32
	FullName             sql.NullString
	Phone                sql.NullString
	Location             sql.NullString
	Headline             sql.NullString
	Summary              sql.NullString
	TotalExperienceYears sql.NullFloat64
	HighestDegree        sql.NullString
}

func (resumeProfileRecord) TableName() string { return "resume_profiles" }

type resumeParseRunRecord struct {
	ID            int64 `gorm:"primaryKey"`
	ResumeID      int64
	UserID        int64
	Status        string
	ParserVersion sql.NullString
	InputHash     sql.NullString
}

func (resumeParseRunRecord) TableName() string { return "resume_parse_runs" }

type resumeEducationRecord struct {
	ID              int64 `gorm:"primaryKey"`
	ResumeProfileID int64
	School          string
	Degree          sql.NullString
	Major           sql.NullString
	StartDate       *time.Time
	EndDate         *time.Time
	Description     sql.NullString
	SortOrder       int32
}

func (resumeEducationRecord) TableName() string { return "resume_educations" }

type resumeExperienceRecord struct {
	ID               int64 `gorm:"primaryKey"`
	ResumeProfileID  int64
	Company          string
	Title            sql.NullString
	Location         sql.NullString
	StartDate        *time.Time
	EndDate          *time.Time
	IsCurrent        int32
	Description      sql.NullString
	AchievementsJSON sql.NullString `gorm:"column:achievements_json"`
	SortOrder        int32
}

func (resumeExperienceRecord) TableName() string { return "resume_experiences" }

type resumeSkillRecord struct {
	ID              int64 `gorm:"primaryKey"`
	ResumeProfileID int64
	Name            string
	SortOrder       int32
}

func (resumeSkillRecord) TableName() string { return "resume_skills" }

type resumeProjectRecord struct {
	ID              int64 `gorm:"primaryKey"`
	ResumeProfileID int64
	Name            string
	SortOrder       int32
}

func (resumeProjectRecord) TableName() string { return "resume_projects" }

func (s *nativeStore) loadProfileBundle(ctx context.Context, userID int64) (profilepkg.Bundle, error) {
	bundle := profilepkg.Bundle{Profile: modelCandidateProfileFromRecord(candidateProfileRecord{UserID: userID})}
	var profile candidateProfileRecord
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		bundle.Profile.UserID = userID
	} else if err != nil {
		return bundle, err
	} else {
		bundle.Profile = modelCandidateProfileFromRecord(profile)
	}
	var educations []candidateEducationRecord
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("sort_order ASC, id ASC").Find(&educations).Error; err != nil {
		return bundle, err
	}
	bundle.Educations = educationsFromRecords(educations)
	var experiences []candidateExperienceRecord
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("sort_order ASC, id ASC").Find(&experiences).Error; err != nil {
		return bundle, err
	}
	bundle.Experiences = experiencesFromRecords(experiences)
	return bundle, nil
}

func (s *nativeStore) saveProfileBundle(ctx context.Context, bundle profilepkg.Bundle) error {
	profilepkg.DenormalizeSummary(&bundle)
	profilepkg.Complete(&bundle)
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing candidateProfileRecord
		err := tx.Where("user_id = ?", bundle.Profile.UserID).First(&existing).Error
		row := recordFromModelProfile(bundle.Profile)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			row.ID = existing.ID
			if err := tx.Model(&candidateProfileRecord{}).Where("id = ?", existing.ID).Updates(map[string]any{
				"real_name": row.RealName, "phone": row.Phone, "education": row.Education, "school": row.School,
				"work_experience": row.WorkExperience, "skills": row.Skills, "city": row.City,
				"years_of_experience": row.YearsOfExperience, "job_status": row.JobStatus,
				"expected_position": row.ExpectedPosition, "expected_salary_min": row.ExpectedSalaryMin,
				"expected_salary_max": row.ExpectedSalaryMax, "available_from": row.AvailableFrom,
				"summary": row.Summary, "is_complete": row.IsComplete,
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("user_id = ?", bundle.Profile.UserID).Delete(&candidateEducationRecord{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", bundle.Profile.UserID).Delete(&candidateExperienceRecord{}).Error; err != nil {
			return err
		}
		for _, edu := range bundle.Educations {
			rec := educationRecordFromInput(bundle.Profile.UserID, edu)
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
		}
		for _, exp := range bundle.Experiences {
			rec := experienceRecordFromInput(bundle.Profile.UserID, exp)
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *nativeStore) backfillLegacyStructuredRows(ctx context.Context, userID int64, profile candidateProfileRecord) error {
	var eduCount int64
	if err := s.db.WithContext(ctx).Model(&candidateEducationRecord{}).Where("user_id = ?", userID).Count(&eduCount).Error; err != nil {
		return err
	}
	if eduCount == 0 && strings.TrimSpace(profile.School) != "" {
		if err := s.db.WithContext(ctx).Create(&candidateEducationRecord{
			UserID:    userID,
			School:    strings.TrimSpace(profile.School),
			Degree:    nullableSQLString(profile.Education),
			SortOrder: 0,
		}).Error; err != nil {
			return err
		}
	}
	var expCount int64
	if err := s.db.WithContext(ctx).Model(&candidateExperienceRecord{}).Where("user_id = ?", userID).Count(&expCount).Error; err != nil {
		return err
	}
	if expCount == 0 && strings.TrimSpace(profile.WorkExperience) != "" {
		if err := s.db.WithContext(ctx).Create(&candidateExperienceRecord{
			UserID:      userID,
			Company:     "历史经历",
			Description: nullableSQLString(profile.WorkExperience),
			SortOrder:   0,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func (s *nativeStore) evaluateFillRefresh(ctx context.Context, userID int64, forceRefresh bool) (resumeID int64, needsRefresh bool, reason string, err error) {
	var resume resumeRecord
	if err := s.db.WithContext(ctx).Where("user_id = ? AND is_valid = ?", userID, 1).Order("uploaded_at DESC, id DESC").First(&resume).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, profilepkg.RefreshReasonNoResume, nil
		}
		return 0, false, "", err
	}
	var profile resumeProfileRecord
	profileErr := s.db.WithContext(ctx).Where("resume_id = ? AND is_current = ?", resume.ID, 1).First(&profile).Error
	hasProfile := profileErr == nil
	if profileErr != nil && !errors.Is(profileErr, gorm.ErrRecordNotFound) {
		return 0, false, "", profileErr
	}
	parserVersion := ""
	inputHash := ""
	if hasProfile && profile.ParseRunID > 0 {
		var run resumeParseRunRecord
		if err := s.db.WithContext(ctx).Where("id = ?", profile.ParseRunID).First(&run).Error; err == nil {
			parserVersion = nullString(run.ParserVersion)
			inputHash = nullString(run.InputHash)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, "", err
		}
	}
	needs, reason := profilepkg.EvaluateResumeFillRefresh(profilepkg.ResumeFillRefreshInput{
		ForceRefresh:    forceRefresh,
		HasResume:       true,
		ParsedText:      resume.ParsedText,
		HasProfile:      hasProfile,
		ParserVersion:   parserVersion,
		StoredInputHash: inputHash,
	})
	return resume.ID, needs, reason, nil
}

func (s *nativeStore) buildFillDraftFromResumeProfile(ctx context.Context, userID int64, overwrite bool) (*pb.ProfileFillDraft, error) {
	var resume resumeRecord
	if err := s.db.WithContext(ctx).Where("user_id = ? AND is_valid = ?", userID, 1).Order("uploaded_at DESC, id DESC").First(&resume).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNoResume
		}
		return nil, err
	}
	if strings.TrimSpace(resume.ParsedText) == "" {
		return nil, errNoParsedText
	}
	var profile resumeProfileRecord
	if err := s.db.WithContext(ctx).Where("resume_id = ? AND is_current = ?", resume.ID, 1).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNoResumeProfile
		}
		return nil, err
	}
	var educations []resumeEducationRecord
	if err := s.db.WithContext(ctx).Where("resume_profile_id = ?", profile.ID).Order("sort_order ASC, id ASC").Find(&educations).Error; err != nil {
		return nil, err
	}
	var experiences []resumeExperienceRecord
	if err := s.db.WithContext(ctx).Where("resume_profile_id = ?", profile.ID).Order("sort_order ASC, id ASC").Find(&experiences).Error; err != nil {
		return nil, err
	}
	var skills []resumeSkillRecord
	if err := s.db.WithContext(ctx).Where("resume_profile_id = ?", profile.ID).Order("sort_order ASC, id ASC").Find(&skills).Error; err != nil {
		return nil, err
	}
	var projectCount int64
	if err := s.db.WithContext(ctx).Model(&resumeProjectRecord{}).Where("resume_profile_id = ?", profile.ID).Count(&projectCount).Error; err != nil {
		return nil, err
	}

	warnings := make([]string, 0, 4)
	draft := &pb.CandidateProfile{}
	if profile.FullName.Valid {
		draft.RealName = strings.TrimSpace(profile.FullName.String)
	}
	if profile.Phone.Valid {
		if phone, ok := profilepkg.NormalizePhone(profile.Phone.String); ok {
			draft.Phone = phone
		} else if strings.TrimSpace(profile.Phone.String) != "" {
			warnings = append(warnings, "简历电话格式无法识别，请手动填写联系电话")
		}
	}
	if profile.Location.Valid {
		if city, ok := profilepkg.NormalizeRegionForProfile(profile.Location.String); ok {
			draft.City = city
			if !profilepkg.IsRegionPathComplete(city) {
				warnings = append(warnings, "所在城市已识别到省/市，请补选区县")
			}
		} else if strings.TrimSpace(profile.Location.String) != "" {
			warnings = append(warnings, "所在城市无法自动规范为省/市/区，请手动选择")
		}
	}
	if profile.Summary.Valid {
		draft.Summary = profilepkg.TruncateRunes(profile.Summary.String, 500)
	}
	if profile.Headline.Valid {
		draft.ExpectedPosition = profilepkg.TruncateRunes(profile.Headline.String, 500)
	}
	if profile.TotalExperienceYears.Valid {
		draft.YearsOfExperience = profile.TotalExperienceYears.Float64
	}
	if profile.HighestDegree.Valid {
		if degree, matched, empty := profilepkg.NormalizeDegree(profile.HighestDegree.String); !empty {
			draft.Education = degree
			if !matched {
				warnings = append(warnings, "最高学历「"+strings.TrimSpace(profile.HighestDegree.String)+"」未匹配标准枚举，已保留原值")
			}
		}
	}
	for _, edu := range educations {
		degree, matched, _ := profilepkg.NormalizeDegree(nullString(edu.Degree))
		if nullString(edu.Degree) != "" && !matched {
			warnings = append(warnings, "教育经历学历「"+nullString(edu.Degree)+"」未匹配标准枚举，已保留原值")
		}
		draft.Educations = append(draft.Educations, &pb.CandidateEducationInfo{
			School: edu.School, Degree: degree, Major: nullString(edu.Major),
			StartDate: formatOptionalDate(edu.StartDate), EndDate: formatOptionalDate(edu.EndDate),
			Description: nullString(edu.Description), SortOrder: edu.SortOrder,
		})
	}
	for _, exp := range experiences {
		location := nullString(exp.Location)
		if location != "" {
			if normalized, ok := profilepkg.NormalizeRegionForProfile(location); ok {
				location = normalized
			} else {
				warnings = append(warnings, "工作经历城市「"+location+"」无法自动规范，请手动选择")
				location = ""
			}
		}
		description := profilepkg.JoinAchievementsIntoDescription(nullString(exp.Description), nullString(exp.AchievementsJSON))
		draft.Experiences = append(draft.Experiences, &pb.CandidateExperienceInfo{
			Company: exp.Company, Title: nullString(exp.Title), Location: location,
			StartDate: formatOptionalDate(exp.StartDate), EndDate: formatOptionalDate(exp.EndDate),
			IsCurrent: exp.IsCurrent, Description: description, SortOrder: exp.SortOrder,
		})
	}
	seenSkills := make(map[string]struct{})
	for _, skill := range skills {
		trimmed := strings.TrimSpace(skill.Name)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seenSkills[key]; ok {
			continue
		}
		seenSkills[key] = struct{}{}
		draft.Skills = append(draft.Skills, trimmed)
	}
	if projectCount > 0 {
		warnings = append(warnings, "简历含 "+strconv.FormatInt(projectCount, 10)+" 个项目经历，暂不写入个人资料")
	}

	existing, err := s.loadProfileBundle(ctx, userID)
	if err != nil {
		return nil, err
	}
	diffs := profilepkg.BuildProfileFillDiffs(existing, profileFillDraftFromPB(draft), overwrite, int(projectCount))
	return &pb.ProfileFillDraft{
		Draft:         draft,
		RefreshReason: profilepkg.RefreshReasonReused,
		FieldDiffs:    profileFillDiffsToPB(diffs),
		Warnings:      uniqueStrings(warnings),
		ResumeId:      resume.ID,
	}, nil
}

func profileFillDraftFromPB(draft *pb.CandidateProfile) profilepkg.ProfileFillDraft {
	if draft == nil {
		return profilepkg.ProfileFillDraft{}
	}
	out := profilepkg.ProfileFillDraft{
		RealName:          draft.GetRealName(),
		Phone:             draft.GetPhone(),
		City:              draft.GetCity(),
		ExpectedPosition:  draft.GetExpectedPosition(),
		Summary:           draft.GetSummary(),
		YearsOfExperience: draft.GetYearsOfExperience(),
		Skills:            append([]string(nil), draft.GetSkills()...),
		Educations:        make([]profilepkg.EducationInput, 0, len(draft.GetEducations())),
		Experiences:       make([]profilepkg.ExperienceInput, 0, len(draft.GetExperiences())),
	}
	for _, item := range draft.GetEducations() {
		if item == nil {
			continue
		}
		out.Educations = append(out.Educations, profilepkg.EducationInput{
			School:      item.GetSchool(),
			Degree:      item.GetDegree(),
			Major:       item.GetMajor(),
			StartDate:   item.GetStartDate(),
			EndDate:     item.GetEndDate(),
			Description: item.GetDescription(),
			SortOrder:   item.GetSortOrder(),
		})
	}
	for _, item := range draft.GetExperiences() {
		if item == nil {
			continue
		}
		out.Experiences = append(out.Experiences, profilepkg.ExperienceInput{
			Company:     item.GetCompany(),
			Title:       item.GetTitle(),
			Location:    item.GetLocation(),
			StartDate:   item.GetStartDate(),
			EndDate:     item.GetEndDate(),
			IsCurrent:   item.GetIsCurrent(),
			Description: item.GetDescription(),
			SortOrder:   item.GetSortOrder(),
		})
	}
	return out
}

func profileFillDiffsToPB(diffs []profilepkg.ProfileFillFieldDiff) []*pb.ProfileFillFieldDiff {
	out := make([]*pb.ProfileFillFieldDiff, 0, len(diffs))
	for _, diff := range diffs {
		out = append(out, &pb.ProfileFillFieldDiff{
			Field:  diff.Field,
			Label:  diff.Label,
			Action: diff.Action,
			Before: diff.Before,
			After:  diff.After,
		})
	}
	return out
}

var (
	errNoResume        = errors.New("no resume")
	errNoResumeProfile = errors.New("no resume profile")
	errNoParsedText    = errors.New("no parsed text")
)

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return values
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func mergeProfileFill(existing profilepkg.Bundle, draft *pb.CandidateProfile, overwrite bool) profilepkg.Bundle {
	if draft == nil {
		return existing
	}
	merged := existing
	setString := func(current *string, incoming string) {
		incoming = strings.TrimSpace(incoming)
		if incoming == "" {
			return
		}
		if overwrite || strings.TrimSpace(*current) == "" {
			*current = incoming
		}
	}
	setString(&merged.Profile.RealName, draft.RealName)
	setString(&merged.Profile.Phone, draft.Phone)
	if shouldReplaceCity(merged.Profile.City, draft.City, overwrite) {
		merged.Profile.City = strings.TrimSpace(draft.City)
	}
	setString(&merged.Profile.JobStatus, draft.JobStatus)
	setString(&merged.Profile.ExpectedPosition, draft.ExpectedPosition)
	setString(&merged.Profile.Summary, draft.Summary)
	setString(&merged.Profile.AvailableFrom, draft.AvailableFrom)
	if draft.YearsOfExperience > 0 && (overwrite || merged.Profile.YearsOfExperience <= 0) {
		merged.Profile.YearsOfExperience = draft.YearsOfExperience
	}
	if draft.ExpectedSalaryMin > 0 && (overwrite || merged.Profile.ExpectedSalaryMin <= 0) {
		merged.Profile.ExpectedSalaryMin = draft.ExpectedSalaryMin
	}
	if draft.ExpectedSalaryMax > 0 && (overwrite || merged.Profile.ExpectedSalaryMax <= 0) {
		merged.Profile.ExpectedSalaryMax = draft.ExpectedSalaryMax
	}
	if len(draft.Skills) > 0 && (overwrite || strings.TrimSpace(merged.Profile.Skills) == "") {
		merged.Profile.Skills = profilepkg.JoinSkills(draft.Skills)
	}
	merged.Educations = profilepkg.MergeEducations(merged.Educations, educationsFromPB(draft.Educations), overwrite)
	merged.Experiences = profilepkg.MergeExperiences(merged.Experiences, experiencesFromPB(draft.Experiences), overwrite)
	return merged
}

func shouldReplaceCity(existing, draft string, overwrite bool) bool {
	draft = strings.TrimSpace(draft)
	if draft == "" {
		return false
	}
	existing = strings.TrimSpace(existing)
	if overwrite || existing == "" {
		return true
	}
	if profilepkg.IsRegionPathComplete(existing) {
		return false
	}
	// Replace free-text / incomplete values with a better slash-separated draft.
	if profilepkg.IsRegionPathComplete(draft) {
		return true
	}
	if strings.Contains(draft, "/") && !strings.Contains(existing, "/") {
		return true
	}
	if strings.Count(draft, "/") > strings.Count(existing, "/") {
		return true
	}
	return draft != existing && strings.Contains(draft, "/")
}

func bundleFromUpdateRequest(req *pb.UpdateProfileRequest) profilepkg.Bundle {
	return profilepkg.Bundle{
		Profile: model.CandidateProfile{
			UserID:            req.UserId,
			RealName:          req.RealName,
			Phone:             req.Phone,
			Education:         req.Education,
			School:            req.School,
			WorkExperience:    req.WorkExperience,
			Skills:            req.Skills,
			City:              req.City,
			YearsOfExperience: req.YearsOfExperience,
			JobStatus:         req.JobStatus,
			ExpectedPosition:  req.ExpectedPosition,
			ExpectedSalaryMin: req.ExpectedSalaryMin,
			ExpectedSalaryMax: req.ExpectedSalaryMax,
			AvailableFrom:     req.AvailableFrom,
			Summary:           req.Summary,
		},
		Educations:  educationsFromPB(req.Educations),
		Experiences: experiencesFromPB(req.Experiences),
	}
}

func modelCandidateProfileFromRecord(row candidateProfileRecord) model.CandidateProfile {
	return model.CandidateProfile{
		UserID: row.UserID, RealName: row.RealName, Phone: row.Phone, Education: row.Education,
		School: row.School, WorkExperience: row.WorkExperience, Skills: row.Skills, City: row.City,
		YearsOfExperience: row.YearsOfExperience, JobStatus: row.JobStatus, ExpectedPosition: row.ExpectedPosition,
		ExpectedSalaryMin: row.ExpectedSalaryMin, ExpectedSalaryMax: row.ExpectedSalaryMax,
		AvailableFrom: formatOptionalDate(row.AvailableFrom), Summary: row.Summary, IsComplete: row.IsComplete,
	}
}

func recordFromModelProfile(profile model.CandidateProfile) candidateProfileRecord {
	return candidateProfileRecord{
		UserID: profile.UserID, RealName: profile.RealName, Phone: profile.Phone, Education: profile.Education,
		School: profile.School, WorkExperience: profile.WorkExperience, Skills: profile.Skills, City: profile.City,
		YearsOfExperience: profile.YearsOfExperience, JobStatus: profile.JobStatus, ExpectedPosition: profile.ExpectedPosition,
		ExpectedSalaryMin: profile.ExpectedSalaryMin, ExpectedSalaryMax: profile.ExpectedSalaryMax,
		AvailableFrom: parseOptionalDate(profile.AvailableFrom), Summary: profile.Summary, IsComplete: profile.IsComplete,
	}
}

func educationsFromPB(items []*pb.CandidateEducationInfo) []profilepkg.EducationInput {
	out := make([]profilepkg.EducationInput, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, profilepkg.EducationInput{
			School: item.School, Degree: item.Degree, Major: item.Major,
			StartDate: item.StartDate, EndDate: item.EndDate, Description: item.Description, SortOrder: item.SortOrder,
		})
	}
	return out
}

func experiencesFromPB(items []*pb.CandidateExperienceInfo) []profilepkg.ExperienceInput {
	out := make([]profilepkg.ExperienceInput, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, profilepkg.ExperienceInput{
			Company: item.Company, Title: item.Title, Location: item.Location,
			StartDate: item.StartDate, EndDate: item.EndDate, IsCurrent: item.IsCurrent,
			Description: item.Description, SortOrder: item.SortOrder,
		})
	}
	return out
}

func educationsFromRecords(items []candidateEducationRecord) []profilepkg.EducationInput {
	out := make([]profilepkg.EducationInput, 0, len(items))
	for _, item := range items {
		out = append(out, profilepkg.EducationInput{
			School: item.School, Degree: nullString(item.Degree), Major: nullString(item.Major),
			StartDate: formatOptionalDate(item.StartDate), EndDate: formatOptionalDate(item.EndDate),
			Description: nullString(item.Description), SortOrder: item.SortOrder,
		})
	}
	return out
}

func experiencesFromRecords(items []candidateExperienceRecord) []profilepkg.ExperienceInput {
	out := make([]profilepkg.ExperienceInput, 0, len(items))
	for _, item := range items {
		out = append(out, profilepkg.ExperienceInput{
			Company: item.Company, Title: nullString(item.Title), Location: nullString(item.Location),
			StartDate: formatOptionalDate(item.StartDate), EndDate: formatOptionalDate(item.EndDate),
			IsCurrent: item.IsCurrent, Description: nullString(item.Description), SortOrder: item.SortOrder,
		})
	}
	return out
}

func educationRecordFromInput(userID int64, item profilepkg.EducationInput) candidateEducationRecord {
	return candidateEducationRecord{
		UserID: userID, School: strings.TrimSpace(item.School), Degree: nullableSQLString(item.Degree),
		Major: nullableSQLString(item.Major), StartDate: parseOptionalDate(item.StartDate),
		EndDate: parseOptionalDate(item.EndDate), Description: nullableSQLString(item.Description), SortOrder: item.SortOrder,
	}
}

func experienceRecordFromInput(userID int64, item profilepkg.ExperienceInput) candidateExperienceRecord {
	return candidateExperienceRecord{
		UserID: userID, Company: strings.TrimSpace(item.Company), Title: nullableSQLString(item.Title),
		Location: nullableSQLString(item.Location), StartDate: parseOptionalDate(item.StartDate),
		EndDate: parseOptionalDate(item.EndDate), IsCurrent: item.IsCurrent,
		Description: nullableSQLString(item.Description), SortOrder: item.SortOrder,
	}
}

func profileBundleToPB(bundle profilepkg.Bundle) *pb.CandidateProfile {
	profile := bundle.Profile
	return &pb.CandidateProfile{
		RealName: profile.RealName, Phone: profile.Phone, Education: profile.Education, School: profile.School,
		WorkExperience: profile.WorkExperience, Skills: splitSkills(profile.Skills), IsComplete: profile.IsComplete == 1,
		City: profile.City, YearsOfExperience: profile.YearsOfExperience, JobStatus: profile.JobStatus,
		ExpectedPosition: profile.ExpectedPosition, ExpectedSalaryMin: profile.ExpectedSalaryMin,
		ExpectedSalaryMax: profile.ExpectedSalaryMax, AvailableFrom: profile.AvailableFrom, Summary: profile.Summary,
		Educations: educationsPBFromInputs(bundle.Educations), Experiences: experiencesPBFromInputs(bundle.Experiences),
	}
}

func educationsPBFromInputs(items []profilepkg.EducationInput) []*pb.CandidateEducationInfo {
	out := make([]*pb.CandidateEducationInfo, 0, len(items))
	for _, item := range items {
		out = append(out, &pb.CandidateEducationInfo{
			School: item.School, Degree: item.Degree, Major: item.Major,
			StartDate: item.StartDate, EndDate: item.EndDate, Description: item.Description, SortOrder: item.SortOrder,
		})
	}
	return out
}

func experiencesPBFromInputs(items []profilepkg.ExperienceInput) []*pb.CandidateExperienceInfo {
	out := make([]*pb.CandidateExperienceInfo, 0, len(items))
	for _, item := range items {
		out = append(out, &pb.CandidateExperienceInfo{
			Company: item.Company, Title: item.Title, Location: item.Location,
			StartDate: item.StartDate, EndDate: item.EndDate, IsCurrent: item.IsCurrent,
			Description: item.Description, SortOrder: item.SortOrder,
		})
	}
	return out
}

func bundleFromProfilepkg(bundle profilepkg.Bundle) profilepkg.Bundle {
	return bundle
}

func parseOptionalDate(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil
	}
	return &parsed
}

func formatOptionalDate(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.Format("2006-01-02")
}

func nullString(value sql.NullString) string {
	if value.Valid {
		return strings.TrimSpace(value.String)
	}
	return ""
}

func nullableSQLString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}
