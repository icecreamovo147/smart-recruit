package port

type ResumeURLSigner interface {
	GeneratePresignedGetURL(ossKey string) (string, error)
}
