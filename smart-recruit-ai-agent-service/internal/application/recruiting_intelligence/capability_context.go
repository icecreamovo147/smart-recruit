package recruiting_intelligence

import "context"

type capabilityRuntimeContext struct {
	modelID           int64
	requestedModelID  int64
	fallbackReason    string
	promptTemplateIDs []int64
	versionID         int64
	snapshotHash      string
}

type capabilityRuntimeContextKey struct{}

func WithCapabilityRuntime(ctx context.Context, requestedModelID, modelID, versionID int64, fallbackReason, snapshotHash string, promptTemplateIDs []int64) context.Context {
	return context.WithValue(ctx, capabilityRuntimeContextKey{}, capabilityRuntimeContext{
		modelID: modelID, requestedModelID: requestedModelID, fallbackReason: fallbackReason, versionID: versionID, snapshotHash: snapshotHash,
		promptTemplateIDs: append([]int64(nil), promptTemplateIDs...),
	})
}

func CapabilityRuntimeModelID(ctx context.Context) int64 {
	value, _ := ctx.Value(capabilityRuntimeContextKey{}).(capabilityRuntimeContext)
	return value.modelID
}

func CapabilityRuntimePromptTemplateIDs(ctx context.Context) []int64 {
	value, _ := ctx.Value(capabilityRuntimeContextKey{}).(capabilityRuntimeContext)
	return append([]int64(nil), value.promptTemplateIDs...)
}

func CapabilityRuntimeTrace(ctx context.Context) (requestedModelID, effectiveModelID, versionID int64, fallbackReason, snapshotHash string) {
	value, _ := ctx.Value(capabilityRuntimeContextKey{}).(capabilityRuntimeContext)
	return value.requestedModelID, value.modelID, value.versionID, value.fallbackReason, value.snapshotHash
}
