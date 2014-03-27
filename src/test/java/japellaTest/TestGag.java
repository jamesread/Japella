package japellaTest;

import japella.Bot;
import japella.MessageParser;
import japella.MessagePlugin.Message;
import japella.messagePlugins.GaggingPlugin;

import org.junit.Assert;
import org.junit.Test;

public class TestGag {
	@Test
	public void testGag() {
		Bot bot = new Bot("gagging victim", null);
		bot.loadMessagePlugin(new GaggingPlugin());
		Message message = bot.onMockMessage(new Message(bot, "#testChannel", "gagger", new MessageParser("!sleep")));

		Assert.assertTrue(bot.isGagged("#testChannel"));
	}

	@Test
	public void testWakeup() {
		Bot bot = new Bot("gagging victim", null);
		bot.loadMessagePlugin(new GaggingPlugin());
		Message message = bot.onMockMessage(new Message(bot, "#testChannel", "gagger", new MessageParser("!wakeup")));

		Assert.assertFalse(bot.isGagged("#testChannel"));
	}
}
