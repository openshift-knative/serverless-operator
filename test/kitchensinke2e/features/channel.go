package features

import (
	"fmt"

	subscriptionresources "knative.dev/eventing/test/rekt/resources/subscription"
	"knative.dev/reconciler-test/pkg/feature"
)

// Prepare lists of kinds to use as channels, subscribers, replies
var channels = []component{
	kafkaChannel,
	inMemoryChannel,
	genericChannelWithKafkaChannelTemplate,
	genericChannelWithInMemoryChannelTemplate,
}

var (
	subscribers      = sinksAll
	subscribersShort = sinksShort
	replies          = sinksAll
	repliesShort     = sinksShort
)

func ChannelReadiness(index int, channel component, subscriber, reply, dls component) *feature.Feature {
	// Make a unique label to use for uniqueness of the subtest and its resources
	testLabel := shortLabel(channel) + shortLabel(subscriber) + shortLabel(reply) + shortLabel(dls)
	testLabel = fmt.Sprintf("%s-%d", testLabel, index)
	channelName := testLabel
	receiverName := testLabel + "-receiver"
	dlsName := testLabel + "-dls"
	replyName := testLabel + "-reply"

	f := feature.NewFeatureNamed(fmt.Sprintf("%s with Subscription to %s with %s replies and %s dls", label(channel), label(subscriber), label(reply), label(dls)))

	f.Setup("Install A Channel", channel.Install(channelName))
	f.Setup("Install A Receiver", subscriber.Install(receiverName))

	// In theory, we could have a "none" component that would have a NOP Install,
	// but we don't control stuff like subscription::WithDeadLetterSink, so we just use nil explicitly to do nothing
	if reply != nil {
		f.Setup("Install A Reply", reply.Install(replyName))
	}

	if dls != nil {
		f.Setup("Install A DLS", dls.Install(dlsName))
	}

	f.Setup("Install A Subscription", subscriptionresources.Install(channelName,
		subscriptionresources.WithChannel(channel.KReference(channelName)),
		subscriptionresources.WithSubscriber(subscriber.KReference(receiverName), ""),
		func(m map[string]interface{}) {
			if reply != nil {
				subscriptionresources.WithReply(reply.KReference(replyName), "")(m)
			}
		},
		func(m map[string]interface{}) {
			if dls != nil {
				subscriptionresources.WithDeadLetterSink(dls.KReference(dlsName), "")(m)
			}
		},
	))

	f.Assert("Channel Is Ready", channel.IsReady(channelName))
	f.Assert("Subscription Is Ready", subscriptionresources.IsReady(channelName))
	return f
}

func ChannelFeatureSet() feature.FeatureSet {
	return channelFeatureSet(false, 1)
}

func ChannelFeatureSetShort() feature.FeatureSet {
	return channelFeatureSet(true, 1)
}

func ChannelFeatureSetStress() feature.FeatureSet {
	return channelFeatureSet(true, NumDeployments)
}

func channelFeatureSet(short bool, times int) feature.FeatureSet {
	dls := deadLetterSinks
	rep := replies
	subscr := subscribers
	if short {
		dls = deadLetterSinksShort
		rep = repliesShort
		subscr = subscribersShort
	}

	type testCombination struct {
		channel        component
		subscriber     component
		reply          component
		deadLetterSink component
	}

	testCombinations := make([]testCombination, 0)

	// Test all combinations of Channel X Subscriber, with no replies or DLS
	for _, channel := range channels {
		for _, subscriber := range subscr {
			testCombinations = append(testCombinations, testCombination{channel: channel, subscriber: subscriber})
		}
	}

	// Test all combinations of Channel X Reply, with any subscriber (we'll use the same subscriber Kind as the reply)
	for _, channel := range channels {
		for _, reply := range rep {
			testCombinations = append(testCombinations, testCombination{channel, reply, reply, nil})
		}
	}

	// Test all combinations of Channel X DLS, with any subscriber (we'll use the same subscriber Kind as the DLS)
	for _, channel := range channels {
		for _, deadLetterSink := range dls {
			testCombinations = append(testCombinations, testCombination{channel, deadLetterSink, nil, deadLetterSink})
		}
	}

	features := make([]*feature.Feature, 0, len(testCombinations))
	for _, testCombination := range testCombinations {
		for i := 0; i < times; i++ {
			features = append(features, ChannelReadiness(i,
				testCombination.channel,
				testCombination.subscriber,
				testCombination.reply,
				testCombination.deadLetterSink,
			))
		}
	}

	return feature.FeatureSet{
		Name:     "Channel",
		Features: features,
	}
}
