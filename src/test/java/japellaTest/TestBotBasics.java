package japellaTest;

import japella.Bot;
import japella.Main;
import japella.MessageParser;
import japella.MessagePlugin.Message;
import japella.Server;
import japella.messagePlugins.HelloWorld;
import japella.messagePlugins.Help;

import java.util.Arrays;

import org.hamcrest.Matchers;
import org.junit.Assert;
import org.junit.Test;

public class TestBotBasics {
	@Test
	public void testAddChannels() {
		Main.instance = new Main();

		Server server = new Server("localhost");
		Bot bot = new Bot("testing bot", server);

		Assert.assertEquals("testing bot", bot.getName());

		Assert.assertThat(Arrays.asList(bot.getChannels()), Matchers.is((Matchers.empty())));

		bot.addAdmin("superuser");

		Assert.assertThat(bot.getAdmins(), Matchers.contains("superuser"));
	}

	@Test
	public void testCommandsWithHighlightComma() {
		Bot bot = new Bot("mybot", null);
		bot.loadMessagePlugin(new HelloWorld());

		Message message;

		message = new Message(bot, "#test", "auser", new MessageParser("mybot, !hello"));
		Assert.assertTrue(message.parser.startsWithUsername());
		Assert.assertEquals("mybot", message.parser.getAlertUsername());
		Assert.assertEquals("!hello", message.parser.getKeyword());

		bot.onMockMessage(message);

		Assert.assertThat(message.replies, Matchers.is(Matchers.not(Matchers.empty())));
		Assert.assertEquals("Oh, hi there.", message.replies.firstElement());
	}

	@Test
	public void testCommandsWithHighlighters() {
		Bot bot = new Bot("mybot", null);
		bot.loadMessagePlugin(new HelloWorld());

		Message message;

		message = new Message(bot, "#test", "auser", new MessageParser("mybot: !hello"));
		Assert.assertTrue(message.parser.startsWithUsername());
		Assert.assertEquals("mybot", message.parser.getAlertUsername());
		Assert.assertEquals("!hello", message.parser.getKeyword());

		bot.onMockMessage(message);

		Assert.assertThat(message.replies, Matchers.is(Matchers.not(Matchers.empty())));
		Assert.assertEquals("Oh, hi there.", message.replies.firstElement());
	}

	@Test
	public void testGetPlugin() {
		Bot bot = new Bot("test", null);
		Help help = new Help();
		bot.loadMessagePlugin(help);

		Assert.assertEquals(help, bot.getMessagePlugin("Help"));
		Assert.assertEquals(1, bot.getMessagePlugins().size());
	}

	@Test
	public void testMessageFromAdmin() {
		Bot bot = new Bot("fred", null);
		bot.addAdmin("James");

		Message message = new Message(bot, "#test", "James", new MessageParser("Hello World"));

		Assert.assertTrue(message.fromAdmin());
	}
}
