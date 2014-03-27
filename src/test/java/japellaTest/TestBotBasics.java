package japellaTest;

import japella.Bot;
import japella.Main;
import japella.MessageParser;
import japella.MessagePlugin.Message;
import japella.Server;

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
	public void testMessageFromAdmin() {
		Bot bot = new Bot("fred", null);
		bot.addAdmin("James");

		Message message = new Message(bot, "#test", "James", new MessageParser("Hello World"));

		Assert.assertTrue(message.fromAdmin());
	}
}
