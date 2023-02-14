package features

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	parallelresources "knative.dev/eventing/test/rekt/resources/parallel"

	sequenceresources "knative.dev/eventing/test/rekt/resources/sequence"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/manifest"
)

var components = []component{
	kafkaChannel,
	inMemoryChannel,
	genericChannelWithKafkaChannelTemplate,
	genericChannelWithInMemoryChannelTemplate,
	inMemoryChannelMtBroker,
	kafkaChannelMtBroker,
	kafkaBroker,
	inMemoryChannelSequence,
	kafkaChannelSequence,
	inMemoryChannelParallel,
	kafkaChannelParallel,
	ksvc,
}

type channelTemplate func() manifest.CfgFn

type flowTestConfiguration struct {
	shortLabel      string
	label           string
	channelTemplate channelTemplate
}

var flowTestConfigurations = []flowTestConfiguration{
	{
		shortLabel:      "imc",
		label:           "InMemoryChannel",
		channelTemplate: withInMemoryChannelTemplate,
	},
	{
		shortLabel:      "kc",
		label:           "KafkaChannel",
		channelTemplate: withKafkaChannelTemplate,
	},
}

func SequenceReadiness(testLabel string, flowTestConfiguration flowTestConfiguration, steps []component, reply component) *feature.Feature {
	sequenceName := testLabel
	stepName := func(step component) string {
		return sequenceName + "-" + shortLabel(step)
	}
	f := feature.NewFeatureNamed(fmt.Sprintf("Sequence with %s channelTemplate", flowTestConfiguration.label))

	stepConfigs := []manifest.CfgFn{
		flowTestConfiguration.channelTemplate(),
	}

	if reply != nil {
		replyName := sequenceName + "-reply-" + shortLabel(reply)
		f.Setup(fmt.Sprintf("Install %s", label(reply)), reply.Install(replyName))
		stepConfigs = append(stepConfigs, sequenceresources.WithReply(reply.KReference(replyName), ""))
	}

	for _, step := range steps {
		stepName := stepName(step)
		f.Setup(fmt.Sprintf("Install %s", label(step)), step.Install(stepName))
		stepConfigs = append(stepConfigs, sequenceresources.WithStep(step.KReference(stepName), ""))
	}

	f.Setup("Install A Sequence", func(ctx context.Context, t feature.T) {
		sequenceresources.Install(sequenceName, stepConfigs...)(ctx, t)
	})

	f.Assert("Sequence is Ready", sequenceresources.IsReady(sequenceName))
	for _, step := range steps {
		stepName := stepName(step)
		f.Assert(fmt.Sprintf("Step %s is Ready", label(step)), step.IsReady(stepName))
	}
	return f
}

func ParallelReadiness(testLabel string, flowTestConfiguration flowTestConfiguration, subscribers []component, replies []component, filters []component, reply component) *feature.Feature {
	parallelName := testLabel
	branchName := func(step component) string {
		return parallelName + "-" + shortLabel(step)
	}
	f := feature.NewFeatureNamed(fmt.Sprintf("Parallel with %s channelTemplate", flowTestConfiguration.label))

	branchConfigs := []manifest.CfgFn{
		func(cfg map[string]interface{}) {
			// Pre-create the array of branches
			numberOfBranches := len(subscribers) + len(replies) + len(filters)
			cfg["branches"] = make([]map[string]interface{}, numberOfBranches)
			for i := 0; i < numberOfBranches; i++ {
				cfg["branches"].([]map[string]interface{})[i] = map[string]interface{}{}
			}
		},
		flowTestConfiguration.channelTemplate(),
	}

	if reply != nil {
		replyName := parallelName + "-reply-" + shortLabel(reply)
		f.Setup(fmt.Sprintf("Install %s", label(reply)), reply.Install(replyName))
		branchConfigs = append(branchConfigs, parallelresources.WithReply(reply.KReference(replyName), ""))
	}

	// Branches with just Subscribers
	for i, subscriber := range subscribers {
		index := i
		subscriberName := branchName(subscriber)
		f.Setup(fmt.Sprintf("Install %s", label(subscriber)), subscriber.Install(subscriberName))
		branchConfigs = append(branchConfigs, parallelresources.WithSubscriberAt(index, subscriber.KReference(subscriberName), ""))
	}

	// Branches with replies, using a random subscriber from above
	for i, reply := range replies {
		index := len(subscribers) + i

		subscriber := subscribers[rand.Intn(len(subscribers))]
		subscriberName := branchName(subscriber)
		replyName := branchName(reply) + "-reply"

		f.Setup(fmt.Sprintf("Install %s Reply", label(reply)), reply.Install(replyName))
		branchConfigs = append(branchConfigs,
			parallelresources.WithSubscriberAt(index, subscriber.KReference(subscriberName), ""),
			parallelresources.WithReplyAt(index, reply.KReference(replyName), ""))
	}

	// Branches with filters, using a random subscriber from above
	for i, filter := range filters {
		index := len(subscribers) + len(replies) + i

		subscriber := subscribers[rand.Intn(len(subscribers))]
		subscriberName := branchName(subscriber)
		filterName := branchName(filter) + "-filter"

		f.Setup(fmt.Sprintf("Install %s Filter", label(filter)), filter.Install(filterName))
		branchConfigs = append(branchConfigs,
			parallelresources.WithSubscriberAt(index, subscriber.KReference(subscriberName), ""),
			parallelresources.WithFilterAt(index, filter.KReference(filterName), ""))
	}

	f.Setup("Install A Parallel", func(ctx context.Context, t feature.T) {
		parallelresources.Install(parallelName, branchConfigs...)(ctx, t)
	})

	f.Assert("Parallel is Ready", parallelresources.IsReady(parallelName))
	for _, subscriber := range subscribers {
		subscriberName := branchName(subscriber)
		f.Assert(fmt.Sprintf("Branch Subscriber %s is Ready", label(subscriber)), subscriber.IsReady(subscriberName))
	}

	for _, reply := range replies {
		replyName := branchName(reply) + "-reply"
		f.Assert(fmt.Sprintf("Branch Reply %s is Ready", label(reply)), reply.IsReady(replyName))
	}

	for _, filter := range filters {
		filterName := branchName(filter) + "-filter"
		f.Assert(fmt.Sprintf("Branch Filter %s is Ready", label(filter)), filter.IsReady(filterName))
	}
	return f
}

// SequenceNoReplyFeatureSet returns sequences with all possible Kinds as steps, with no reply
func SequenceNoReplyFeatureSet() feature.FeatureSet {
	var features []*feature.Feature
	steps := components
	for _, flowTestConfiguration := range flowTestConfigurations {
		features = append(features, SequenceReadiness(flowTestConfiguration.shortLabel+"-seq", flowTestConfiguration, steps, nil))
	}
	return feature.FeatureSet{
		Name:     "SequenceNoReply",
		Features: features,
	}
}

// ParallelNoReplyFeatureSet returns parallels with
// * all possible kinds as branches' subscribers,
// * all possible kinds of replies (with a random subscriber each)
// * all possible kind of filters (with a random subscriber each)
// with no global reply
func ParallelNoReplyFeatureSet() feature.FeatureSet {
	var features []*feature.Feature
	filters := components
	for _, flowTestConfiguration := range flowTestConfigurations {
		features = append(features, ParallelReadiness(flowTestConfiguration.shortLabel+"-par", flowTestConfiguration, subscribers, replies, filters, nil))
	}
	return feature.FeatureSet{
		Name:     "ParallelNoReply",
		Features: features,
	}
}

// SequenceGlobalReplyFeatureSet returns sequences with a global reply (with a single random Step)
func SequenceGlobalReplyFeatureSet() feature.FeatureSet {
	// We're using random to choose a random subscriber for a given reply Kind
	rand.Seed(time.Now().Unix())
	var features []*feature.Feature
	steps := components
	for _, flowTestConfiguration := range flowTestConfigurations {
		for _, reply := range replies {
			label := fmt.Sprintf("%s-seq-%s-rep", flowTestConfiguration.shortLabel, shortLabel(reply))
			// We've already tested all possible step kinds above,
			// so just use a single step (with a random step) in the sequence for the "with-reply" test
			features = append(features, SequenceReadiness(label, flowTestConfiguration, []component{steps[rand.Intn(len(steps))]}, reply))
		}
	}
	return feature.FeatureSet{
		Name:     "SequenceGlobalReply",
		Features: features,
	}
}

// ParallelGlobalReplyFeatureSet returns parallels with a global reply (with a single Branch of a random subscriber)
func ParallelGlobalReplyFeatureSet() feature.FeatureSet {
	// TODO: Call this only once?
	// We're using random to choose a random subscriber for a given reply Kind
	rand.Seed(time.Now().Unix())
	var features []*feature.Feature
	for _, flowTestConfiguration := range flowTestConfigurations {
		for _, reply := range replies {
			label := fmt.Sprintf("%s-par-%s-rep", flowTestConfiguration.shortLabel, shortLabel(reply))
			// We've already tested all possible branches kinds above,
			// so just use a single branch (with a random subscriber) in the sequence for the "with-reply" test
			features = append(features, ParallelReadiness(label, flowTestConfiguration, []component{subscribers[rand.Intn(len(subscribers))]}, []component{}, []component{}, reply))
		}
	}
	return feature.FeatureSet{
		Name:     "ParallelGlobalReply",
		Features: features,
	}
}
