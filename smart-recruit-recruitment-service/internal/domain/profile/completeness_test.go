package profile_test

import (
	"testing"

	"smart-recruit-recruitment-service/internal/domain/model"
	profilepkg "smart-recruit-recruitment-service/internal/domain/profile"
)

func TestCompleteProfileRequiresScreeningAndStructuredRows(t *testing.T) {
	bundle := profilepkg.Bundle{
		Profile: model.CandidateProfile{
			RealName: "Ada", Phone: "13800000000", Skills: "Go",
			City: "上海", JobStatus: profilepkg.JobStatusEmployed,
			ExpectedPosition: "后端工程师", Summary: "5年Go经验",
		},
		Educations: []profilepkg.EducationInput{{School: "THU", Degree: "本科"}},
		Experiences: []profilepkg.ExperienceInput{{Company: "Acme", Title: "工程师"}},
	}
	profilepkg.Complete(&bundle)
	if bundle.Profile.IsComplete != 1 {
		t.Fatalf("IsComplete = %d, want 1", bundle.Profile.IsComplete)
	}
}

func TestCompleteProfileAllowsStudentWithoutExperience(t *testing.T) {
	bundle := profilepkg.Bundle{
		Profile: model.CandidateProfile{
			RealName: "Bob", Phone: "13800000001", Skills: "Java",
			City: "北京", JobStatus: profilepkg.JobStatusStudent,
			ExpectedPosition: "实习生", Summary: "计算机专业在读",
		},
		Educations: []profilepkg.EducationInput{{School: "PKU", Degree: "本科"}},
	}
	profilepkg.Complete(&bundle)
	if bundle.Profile.IsComplete != 1 {
		t.Fatalf("IsComplete = %d, want 1", bundle.Profile.IsComplete)
	}
}

func TestCompleteProfileAllowsEmployedWithoutExperience(t *testing.T) {
	bundle := profilepkg.Bundle{
		Profile: model.CandidateProfile{
			RealName: "Ada", Phone: "13800000000", Skills: "Go",
			City: "上海", JobStatus: profilepkg.JobStatusEmployed,
			ExpectedPosition: "后端工程师", Summary: "校招候选人", YearsOfExperience: 0,
		},
		Educations: []profilepkg.EducationInput{{School: "THU", Degree: "本科"}},
	}
	profilepkg.Complete(&bundle)
	if bundle.Profile.IsComplete != 1 {
		t.Fatalf("IsComplete = %d, want 1", bundle.Profile.IsComplete)
	}
}

func TestDenormalizeSummaryWritesFlatFields(t *testing.T) {
	bundle := profilepkg.Bundle{
		Educations: []profilepkg.EducationInput{{School: "THU", Degree: "硕士"}},
		Experiences: []profilepkg.ExperienceInput{{Company: "Acme", Title: "工程师", Description: "负责后端"}},
	}
	profilepkg.DenormalizeSummary(&bundle)
	if bundle.Profile.School != "THU" || bundle.Profile.Education != "硕士" {
		t.Fatalf("education summary = %s/%s", bundle.Profile.Education, bundle.Profile.School)
	}
	if bundle.Profile.WorkExperience == "" {
		t.Fatalf("work experience summary should not be empty")
	}
}
