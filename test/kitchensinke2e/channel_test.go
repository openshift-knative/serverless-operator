package kitchensinke2e

import (
	"context"
	"fmt"
	"testing"

	subscriptionresources "knative.dev/eventing/test/rekt/resources/subscription"
	"knative.dev/reconciler-test/pkg/feature"
	"knative.dev/reconciler-test/pkg/state"
)

const (
	channelNameKey  = "channelName"
	receiverNameKey = "receiverName"
	dlsNameKey      = "dlsName"
	replyNameKey    = "replyName"
)

func ChannelReadiness(ctx context.Context, t *testing.T, channel component, subscriber, reply, dls component) *feature.Feature {

	channelName := state.GetStringOrFail(ctx, t, channelNameKey)
	receiverName := state.GetStringOrFail(ctx, t, receiverNameKey)
	dlsName := state.GetStringOrFail(ctx, t, dlsNameKey)
	replyName := state.GetStringOrFail(ctx, t, replyNameKey)
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

func TestChannelReadiness(t *testing.T) {

	// Prepare lists of kinds to use as channels, subscribers, replies and DLSs
	channels := []component{
		kafkaChannel,
		inMemoryChannel,
		genericChannelWithKafkaChannelTemplate,
		genericChannelWithInMemoryChannelTemplate,
	}

	subscribers := []component{
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

	replies := []component{
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

	deadLetterSinks := []component{
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

	type testCombination struct {
		channel        component
		subscriber     component
		reply          component
		deadLetterSink component
	}

	testCombinations := make([]testCombination, 0)

	// Test all combinations of Channel X Subscriber, with no replies or DLS
	for _, channel := range channels {
		for _, subscriber := range subscribers {
			testCombinations = append(testCombinations, testCombination{channel: channel, subscriber: subscriber})
		}
	}

	// Test all combinations of Channel X Reply, with any subscriber (we'll use the same subscriber Kind as the reply)
	for _, channel := range channels {
		for _, reply := range replies {
			testCombinations = append(testCombinations, testCombination{channel, reply, reply, nil})
		}
	}

	// Test all combinations of Channel X DLS, with any subscriber (we'll use the same subscriber Kind as the DLS)
	for _, channel := range channels {
		for _, deadLetterSink := range deadLetterSinks {
			testCombinations = append(testCombinations, testCombination{channel, deadLetterSink, nil, deadLetterSink})
		}
	}

	for _, testCombination := range testCombinations {
		// We'll run the tests in parallel, so make sure we use variables safely
		ch := testCombination.channel
		subscriber := testCombination.subscriber
		reply := testCombination.reply
		deadLetterSink := testCombination.deadLetterSink

		// We make a unique label to use for uniqueness of the subtest and its resources
		testLabel := shortLabel(ch) + shortLabel(subscriber) + shortLabel(reply) + shortLabel(deadLetterSink)
		t.Run(testLabel, func(t *testing.T) {
			t.Parallel()

			ctx, env := defaultContext(t)

			ctx = state.ContextWith(ctx, &state.KVStore{})

			state.SetOrFail(ctx, t, channelNameKey, testLabel)
			state.SetOrFail(ctx, t, receiverNameKey, testLabel+"-receiver")
			state.SetOrFail(ctx, t, dlsNameKey, testLabel+"-dls")
			state.SetOrFail(ctx, t, replyNameKey, testLabel+"-reply")

			env.Test(ctx, t, ChannelReadiness(ctx, t, ch, subscriber, reply, deadLetterSink))
		})
	}
}
