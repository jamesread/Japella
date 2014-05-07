package japellaTest;

import japella.Bot;
import japella.MessageParser;
import japella.MessagePlugin.Message;
import japella.messagePlugins.Decide;
import japellaTest.mocks.MockRandomBooleans;

import org.hamcrest.Matchers;
import org.junit.Assert;
import org.junit.Test;

public class TestDecide {
	@Test
	public void testDecide() {
		Decide decider = new Decide();

		Bot bot = new Bot("test", null);
		bot.loadMessagePlugin(decider);

		Decide.randomGenerator = new MockRandomBooleans(true);
		Message messageYep = new Message(bot, "testchannel", "testsender", new MessageParser("!decide hello"));
		String replyYep = bot.onMockMessage(messageYep).getReplies().firstElement();

		Assert.assertThat(replyYep, Matchers.startsWith("Yep"));
		Assert.assertThat(replyYep, Matchers.containsString(messageYep.parser.getContentBody()));

		Decide.randomGenerator = new MockRandomBooleans(false);
		Message messageNope = new Message(bot, "testchannel", "testsender", new MessageParser("!decide hello"));
		String replyNope = bot.onMockMessage(messageNope).getReplies().firstElement();

		Assert.assertThat(replyNope, Matchers.startsWith("Nope"));
		Assert.assertThat(replyNope, Matchers.containsString(messageNope.parser.getContentBody()));
	}
}
