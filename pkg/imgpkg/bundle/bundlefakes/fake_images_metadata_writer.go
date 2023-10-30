// Code generated by counterfeiter. DO NOT EDIT.
package bundlefakes

import (
	"sync"

	"carvel.dev/imgpkg/pkg/imgpkg/bundle"
	"carvel.dev/imgpkg/pkg/imgpkg/internal/util"
	"carvel.dev/imgpkg/pkg/imgpkg/registry"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type FakeImagesMetadataWriter struct {
	CloneWithLoggerStub        func(util.ProgressLogger) registry.Registry
	cloneWithLoggerMutex       sync.RWMutex
	cloneWithLoggerArgsForCall []struct {
		arg1 util.ProgressLogger
	}
	cloneWithLoggerReturns struct {
		result1 registry.Registry
	}
	cloneWithLoggerReturnsOnCall map[int]struct {
		result1 registry.Registry
	}
	DigestStub        func(name.Reference) (v1.Hash, error)
	digestMutex       sync.RWMutex
	digestArgsForCall []struct {
		arg1 name.Reference
	}
	digestReturns struct {
		result1 v1.Hash
		result2 error
	}
	digestReturnsOnCall map[int]struct {
		result1 v1.Hash
		result2 error
	}
	FirstImageExistsStub        func([]string) (string, error)
	firstImageExistsMutex       sync.RWMutex
	firstImageExistsArgsForCall []struct {
		arg1 []string
	}
	firstImageExistsReturns struct {
		result1 string
		result2 error
	}
	firstImageExistsReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	GetStub        func(name.Reference) (*remote.Descriptor, error)
	getMutex       sync.RWMutex
	getArgsForCall []struct {
		arg1 name.Reference
	}
	getReturns struct {
		result1 *remote.Descriptor
		result2 error
	}
	getReturnsOnCall map[int]struct {
		result1 *remote.Descriptor
		result2 error
	}
	ImageStub        func(name.Reference) (v1.Image, error)
	imageMutex       sync.RWMutex
	imageArgsForCall []struct {
		arg1 name.Reference
	}
	imageReturns struct {
		result1 v1.Image
		result2 error
	}
	imageReturnsOnCall map[int]struct {
		result1 v1.Image
		result2 error
	}
	WriteImageStub        func(name.Reference, v1.Image, chan v1.Update) error
	writeImageMutex       sync.RWMutex
	writeImageArgsForCall []struct {
		arg1 name.Reference
		arg2 v1.Image
		arg3 chan v1.Update
	}
	writeImageReturns struct {
		result1 error
	}
	writeImageReturnsOnCall map[int]struct {
		result1 error
	}
	WriteTagStub        func(name.Tag, remote.Taggable) error
	writeTagMutex       sync.RWMutex
	writeTagArgsForCall []struct {
		arg1 name.Tag
		arg2 remote.Taggable
	}
	writeTagReturns struct {
		result1 error
	}
	writeTagReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeImagesMetadataWriter) CloneWithLogger(arg1 util.ProgressLogger) registry.Registry {
	fake.cloneWithLoggerMutex.Lock()
	ret, specificReturn := fake.cloneWithLoggerReturnsOnCall[len(fake.cloneWithLoggerArgsForCall)]
	fake.cloneWithLoggerArgsForCall = append(fake.cloneWithLoggerArgsForCall, struct {
		arg1 util.ProgressLogger
	}{arg1})
	stub := fake.CloneWithLoggerStub
	fakeReturns := fake.cloneWithLoggerReturns
	fake.recordInvocation("CloneWithLogger", []interface{}{arg1})
	fake.cloneWithLoggerMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeImagesMetadataWriter) CloneWithLoggerCallCount() int {
	fake.cloneWithLoggerMutex.RLock()
	defer fake.cloneWithLoggerMutex.RUnlock()
	return len(fake.cloneWithLoggerArgsForCall)
}

func (fake *FakeImagesMetadataWriter) CloneWithLoggerCalls(stub func(util.ProgressLogger) registry.Registry) {
	fake.cloneWithLoggerMutex.Lock()
	defer fake.cloneWithLoggerMutex.Unlock()
	fake.CloneWithLoggerStub = stub
}

func (fake *FakeImagesMetadataWriter) CloneWithLoggerArgsForCall(i int) util.ProgressLogger {
	fake.cloneWithLoggerMutex.RLock()
	defer fake.cloneWithLoggerMutex.RUnlock()
	argsForCall := fake.cloneWithLoggerArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeImagesMetadataWriter) CloneWithLoggerReturns(result1 registry.Registry) {
	fake.cloneWithLoggerMutex.Lock()
	defer fake.cloneWithLoggerMutex.Unlock()
	fake.CloneWithLoggerStub = nil
	fake.cloneWithLoggerReturns = struct {
		result1 registry.Registry
	}{result1}
}

func (fake *FakeImagesMetadataWriter) CloneWithLoggerReturnsOnCall(i int, result1 registry.Registry) {
	fake.cloneWithLoggerMutex.Lock()
	defer fake.cloneWithLoggerMutex.Unlock()
	fake.CloneWithLoggerStub = nil
	if fake.cloneWithLoggerReturnsOnCall == nil {
		fake.cloneWithLoggerReturnsOnCall = make(map[int]struct {
			result1 registry.Registry
		})
	}
	fake.cloneWithLoggerReturnsOnCall[i] = struct {
		result1 registry.Registry
	}{result1}
}

func (fake *FakeImagesMetadataWriter) Digest(arg1 name.Reference) (v1.Hash, error) {
	fake.digestMutex.Lock()
	ret, specificReturn := fake.digestReturnsOnCall[len(fake.digestArgsForCall)]
	fake.digestArgsForCall = append(fake.digestArgsForCall, struct {
		arg1 name.Reference
	}{arg1})
	stub := fake.DigestStub
	fakeReturns := fake.digestReturns
	fake.recordInvocation("Digest", []interface{}{arg1})
	fake.digestMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeImagesMetadataWriter) DigestCallCount() int {
	fake.digestMutex.RLock()
	defer fake.digestMutex.RUnlock()
	return len(fake.digestArgsForCall)
}

func (fake *FakeImagesMetadataWriter) DigestCalls(stub func(name.Reference) (v1.Hash, error)) {
	fake.digestMutex.Lock()
	defer fake.digestMutex.Unlock()
	fake.DigestStub = stub
}

func (fake *FakeImagesMetadataWriter) DigestArgsForCall(i int) name.Reference {
	fake.digestMutex.RLock()
	defer fake.digestMutex.RUnlock()
	argsForCall := fake.digestArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeImagesMetadataWriter) DigestReturns(result1 v1.Hash, result2 error) {
	fake.digestMutex.Lock()
	defer fake.digestMutex.Unlock()
	fake.DigestStub = nil
	fake.digestReturns = struct {
		result1 v1.Hash
		result2 error
	}{result1, result2}
}

func (fake *FakeImagesMetadataWriter) DigestReturnsOnCall(i int, result1 v1.Hash, result2 error) {
	fake.digestMutex.Lock()
	defer fake.digestMutex.Unlock()
	fake.DigestStub = nil
	if fake.digestReturnsOnCall == nil {
		fake.digestReturnsOnCall = make(map[int]struct {
			result1 v1.Hash
			result2 error
		})
	}
	fake.digestReturnsOnCall[i] = struct {
		result1 v1.Hash
		result2 error
	}{result1, result2}
}

func (fake *FakeImagesMetadataWriter) FirstImageExists(arg1 []string) (string, error) {
	var arg1Copy []string
	if arg1 != nil {
		arg1Copy = make([]string, len(arg1))
		copy(arg1Copy, arg1)
	}
	fake.firstImageExistsMutex.Lock()
	ret, specificReturn := fake.firstImageExistsReturnsOnCall[len(fake.firstImageExistsArgsForCall)]
	fake.firstImageExistsArgsForCall = append(fake.firstImageExistsArgsForCall, struct {
		arg1 []string
	}{arg1Copy})
	stub := fake.FirstImageExistsStub
	fakeReturns := fake.firstImageExistsReturns
	fake.recordInvocation("FirstImageExists", []interface{}{arg1Copy})
	fake.firstImageExistsMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeImagesMetadataWriter) FirstImageExistsCallCount() int {
	fake.firstImageExistsMutex.RLock()
	defer fake.firstImageExistsMutex.RUnlock()
	return len(fake.firstImageExistsArgsForCall)
}

func (fake *FakeImagesMetadataWriter) FirstImageExistsCalls(stub func([]string) (string, error)) {
	fake.firstImageExistsMutex.Lock()
	defer fake.firstImageExistsMutex.Unlock()
	fake.FirstImageExistsStub = stub
}

func (fake *FakeImagesMetadataWriter) FirstImageExistsArgsForCall(i int) []string {
	fake.firstImageExistsMutex.RLock()
	defer fake.firstImageExistsMutex.RUnlock()
	argsForCall := fake.firstImageExistsArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeImagesMetadataWriter) FirstImageExistsReturns(result1 string, result2 error) {
	fake.firstImageExistsMutex.Lock()
	defer fake.firstImageExistsMutex.Unlock()
	fake.FirstImageExistsStub = nil
	fake.firstImageExistsReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeImagesMetadataWriter) FirstImageExistsReturnsOnCall(i int, result1 string, result2 error) {
	fake.firstImageExistsMutex.Lock()
	defer fake.firstImageExistsMutex.Unlock()
	fake.FirstImageExistsStub = nil
	if fake.firstImageExistsReturnsOnCall == nil {
		fake.firstImageExistsReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.firstImageExistsReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeImagesMetadataWriter) Get(arg1 name.Reference) (*remote.Descriptor, error) {
	fake.getMutex.Lock()
	ret, specificReturn := fake.getReturnsOnCall[len(fake.getArgsForCall)]
	fake.getArgsForCall = append(fake.getArgsForCall, struct {
		arg1 name.Reference
	}{arg1})
	stub := fake.GetStub
	fakeReturns := fake.getReturns
	fake.recordInvocation("Get", []interface{}{arg1})
	fake.getMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeImagesMetadataWriter) GetCallCount() int {
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	return len(fake.getArgsForCall)
}

func (fake *FakeImagesMetadataWriter) GetCalls(stub func(name.Reference) (*remote.Descriptor, error)) {
	fake.getMutex.Lock()
	defer fake.getMutex.Unlock()
	fake.GetStub = stub
}

func (fake *FakeImagesMetadataWriter) GetArgsForCall(i int) name.Reference {
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	argsForCall := fake.getArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeImagesMetadataWriter) GetReturns(result1 *remote.Descriptor, result2 error) {
	fake.getMutex.Lock()
	defer fake.getMutex.Unlock()
	fake.GetStub = nil
	fake.getReturns = struct {
		result1 *remote.Descriptor
		result2 error
	}{result1, result2}
}

func (fake *FakeImagesMetadataWriter) GetReturnsOnCall(i int, result1 *remote.Descriptor, result2 error) {
	fake.getMutex.Lock()
	defer fake.getMutex.Unlock()
	fake.GetStub = nil
	if fake.getReturnsOnCall == nil {
		fake.getReturnsOnCall = make(map[int]struct {
			result1 *remote.Descriptor
			result2 error
		})
	}
	fake.getReturnsOnCall[i] = struct {
		result1 *remote.Descriptor
		result2 error
	}{result1, result2}
}

func (fake *FakeImagesMetadataWriter) Image(arg1 name.Reference) (v1.Image, error) {
	fake.imageMutex.Lock()
	ret, specificReturn := fake.imageReturnsOnCall[len(fake.imageArgsForCall)]
	fake.imageArgsForCall = append(fake.imageArgsForCall, struct {
		arg1 name.Reference
	}{arg1})
	stub := fake.ImageStub
	fakeReturns := fake.imageReturns
	fake.recordInvocation("Image", []interface{}{arg1})
	fake.imageMutex.Unlock()
	if stub != nil {
		return stub(arg1)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeImagesMetadataWriter) ImageCallCount() int {
	fake.imageMutex.RLock()
	defer fake.imageMutex.RUnlock()
	return len(fake.imageArgsForCall)
}

func (fake *FakeImagesMetadataWriter) ImageCalls(stub func(name.Reference) (v1.Image, error)) {
	fake.imageMutex.Lock()
	defer fake.imageMutex.Unlock()
	fake.ImageStub = stub
}

func (fake *FakeImagesMetadataWriter) ImageArgsForCall(i int) name.Reference {
	fake.imageMutex.RLock()
	defer fake.imageMutex.RUnlock()
	argsForCall := fake.imageArgsForCall[i]
	return argsForCall.arg1
}

func (fake *FakeImagesMetadataWriter) ImageReturns(result1 v1.Image, result2 error) {
	fake.imageMutex.Lock()
	defer fake.imageMutex.Unlock()
	fake.ImageStub = nil
	fake.imageReturns = struct {
		result1 v1.Image
		result2 error
	}{result1, result2}
}

func (fake *FakeImagesMetadataWriter) ImageReturnsOnCall(i int, result1 v1.Image, result2 error) {
	fake.imageMutex.Lock()
	defer fake.imageMutex.Unlock()
	fake.ImageStub = nil
	if fake.imageReturnsOnCall == nil {
		fake.imageReturnsOnCall = make(map[int]struct {
			result1 v1.Image
			result2 error
		})
	}
	fake.imageReturnsOnCall[i] = struct {
		result1 v1.Image
		result2 error
	}{result1, result2}
}

func (fake *FakeImagesMetadataWriter) WriteImage(arg1 name.Reference, arg2 v1.Image, arg3 chan v1.Update) error {
	fake.writeImageMutex.Lock()
	ret, specificReturn := fake.writeImageReturnsOnCall[len(fake.writeImageArgsForCall)]
	fake.writeImageArgsForCall = append(fake.writeImageArgsForCall, struct {
		arg1 name.Reference
		arg2 v1.Image
		arg3 chan v1.Update
	}{arg1, arg2, arg3})
	stub := fake.WriteImageStub
	fakeReturns := fake.writeImageReturns
	fake.recordInvocation("WriteImage", []interface{}{arg1, arg2, arg3})
	fake.writeImageMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeImagesMetadataWriter) WriteImageCallCount() int {
	fake.writeImageMutex.RLock()
	defer fake.writeImageMutex.RUnlock()
	return len(fake.writeImageArgsForCall)
}

func (fake *FakeImagesMetadataWriter) WriteImageCalls(stub func(name.Reference, v1.Image, chan v1.Update) error) {
	fake.writeImageMutex.Lock()
	defer fake.writeImageMutex.Unlock()
	fake.WriteImageStub = stub
}

func (fake *FakeImagesMetadataWriter) WriteImageArgsForCall(i int) (name.Reference, v1.Image, chan v1.Update) {
	fake.writeImageMutex.RLock()
	defer fake.writeImageMutex.RUnlock()
	argsForCall := fake.writeImageArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeImagesMetadataWriter) WriteImageReturns(result1 error) {
	fake.writeImageMutex.Lock()
	defer fake.writeImageMutex.Unlock()
	fake.WriteImageStub = nil
	fake.writeImageReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeImagesMetadataWriter) WriteImageReturnsOnCall(i int, result1 error) {
	fake.writeImageMutex.Lock()
	defer fake.writeImageMutex.Unlock()
	fake.WriteImageStub = nil
	if fake.writeImageReturnsOnCall == nil {
		fake.writeImageReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.writeImageReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeImagesMetadataWriter) WriteTag(arg1 name.Tag, arg2 remote.Taggable) error {
	fake.writeTagMutex.Lock()
	ret, specificReturn := fake.writeTagReturnsOnCall[len(fake.writeTagArgsForCall)]
	fake.writeTagArgsForCall = append(fake.writeTagArgsForCall, struct {
		arg1 name.Tag
		arg2 remote.Taggable
	}{arg1, arg2})
	stub := fake.WriteTagStub
	fakeReturns := fake.writeTagReturns
	fake.recordInvocation("WriteTag", []interface{}{arg1, arg2})
	fake.writeTagMutex.Unlock()
	if stub != nil {
		return stub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1
	}
	return fakeReturns.result1
}

func (fake *FakeImagesMetadataWriter) WriteTagCallCount() int {
	fake.writeTagMutex.RLock()
	defer fake.writeTagMutex.RUnlock()
	return len(fake.writeTagArgsForCall)
}

func (fake *FakeImagesMetadataWriter) WriteTagCalls(stub func(name.Tag, remote.Taggable) error) {
	fake.writeTagMutex.Lock()
	defer fake.writeTagMutex.Unlock()
	fake.WriteTagStub = stub
}

func (fake *FakeImagesMetadataWriter) WriteTagArgsForCall(i int) (name.Tag, remote.Taggable) {
	fake.writeTagMutex.RLock()
	defer fake.writeTagMutex.RUnlock()
	argsForCall := fake.writeTagArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeImagesMetadataWriter) WriteTagReturns(result1 error) {
	fake.writeTagMutex.Lock()
	defer fake.writeTagMutex.Unlock()
	fake.WriteTagStub = nil
	fake.writeTagReturns = struct {
		result1 error
	}{result1}
}

func (fake *FakeImagesMetadataWriter) WriteTagReturnsOnCall(i int, result1 error) {
	fake.writeTagMutex.Lock()
	defer fake.writeTagMutex.Unlock()
	fake.WriteTagStub = nil
	if fake.writeTagReturnsOnCall == nil {
		fake.writeTagReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.writeTagReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *FakeImagesMetadataWriter) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.cloneWithLoggerMutex.RLock()
	defer fake.cloneWithLoggerMutex.RUnlock()
	fake.digestMutex.RLock()
	defer fake.digestMutex.RUnlock()
	fake.firstImageExistsMutex.RLock()
	defer fake.firstImageExistsMutex.RUnlock()
	fake.getMutex.RLock()
	defer fake.getMutex.RUnlock()
	fake.imageMutex.RLock()
	defer fake.imageMutex.RUnlock()
	fake.writeImageMutex.RLock()
	defer fake.writeImageMutex.RUnlock()
	fake.writeTagMutex.RLock()
	defer fake.writeTagMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeImagesMetadataWriter) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ bundle.ImagesMetadataWriter = new(FakeImagesMetadataWriter)
