package japellaTest;

import japella.Bot;
import japella.Main;
import japella.MessageParser;
import japella.MessagePlugin.Message;
import japella.Server;
import japella.messagePlugins.Help;

import org.hamcrest.Matchers;
import org.junit.Assert;
import org.junit.Test;

public class TestHelp {
	@Test
	public void testHelpVersion() {
		Main.instance = new Main();
		Bot bot = new Bot("test", new Server("localhost"));
		bot.loadMessagePlugin(new Help());

		Message message = new Message(bot, "testchannel", "testsender", new MessageParser("!help"));
		bot.onMockMessage(message);

		Assert.assertThat(message.getReplies().firstElement(), Matchers.startsWith("Hi. I support these commands"));
	}

	@Test
	public void testPlugins() {
		Bot bot = new Bot("test", null);
		bot.loadMessagePlugin(new Help());

		Message message = new Message(bot, "testchannel", "testsender", new MessageParser("!plugins"));
		bot.onMockMessage(message);

		Assert.assertEquals("Plugins: Help.", message.getReplies().firstElement());
	}

	@Test
	public void testSave() {
		Help help = new Help();
		help.saveConfig();
	}
}
