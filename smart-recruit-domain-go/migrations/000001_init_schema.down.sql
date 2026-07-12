-- Down: 000001_init_schema
-- WARNING: This drops ALL base tables. Data loss is irreversible.
-- Tables must be dropped in reverse dependency order.

DROP TABLE IF EXISTS `ai_memories`;
DROP TABLE IF EXISTS `ai_tool_traces`;
DROP TABLE IF EXISTS `ai_session_summaries`;
DROP TABLE IF EXISTS `ai_chat_history`;
DROP TABLE IF EXISTS `ai_chat_sessions`;
DROP TABLE IF EXISTS `notifications`;
DROP TABLE IF EXISTS `applications`;
DROP TABLE IF EXISTS `resumes`;
DROP TABLE IF EXISTS `candidate_profiles`;
DROP TABLE IF EXISTS `jobs`;
DROP TABLE IF EXISTS `users`;
