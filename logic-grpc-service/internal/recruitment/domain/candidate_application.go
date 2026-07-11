package domain

// CandidateProfileID identifies Recruitment-owned candidate profile data.
type CandidateProfileID int64

// CandidateUserID identifies the candidate account whose recruitment facts are owned here.
type CandidateUserID int64

// ResumeID identifies a Recruitment-owned uploaded resume record.
type ResumeID int64

// ApplicationID identifies a Recruitment-owned application aggregate.
type ApplicationID int64

// ApplicationRound identifies one candidate-to-job application round.
type ApplicationRound int32

// RecruitmentOwnsCandidateFacts documents the source-of-truth boundary for candidate recruitment data.
const RecruitmentOwnsCandidateFacts = "candidate_profile_resume_application"

// AIAgentOwnsDerivedIntelligence documents that AI-derived artifacts remain outside Recruitment ownership.
const AIAgentOwnsDerivedIntelligence = "ai_profile_embedding_matching_memory_intelligence"
