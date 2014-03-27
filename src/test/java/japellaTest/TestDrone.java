package japellaTest;

import japella.Bot;
import japella.MessagePlugin.Message;
import japella.messagePlugins.Drone;

import org.junit.Assert;
import org.junit.Test;

public class TestDrone {
	@Test
	public void testDroneNotInChannel() {
		Bot bot = new Bot("tester", null);
		bot.loadMessagePlugin(new Drone());

		Message message = bot.onMockMessage(new Message(bot, "#foo", "tester", "!say #foo hello there!"));

		Assert.assertEquals("I'm not in the channel: #foo", message.replies.firstElement());
	}
}
