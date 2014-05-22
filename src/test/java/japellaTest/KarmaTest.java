package japellaTest;

import japella.Bot;
import japella.MessageParser;
import japella.MessagePlugin.Message;
import japella.messagePlugins.KarmaTracker;

import org.junit.Assert;
import org.junit.Ignore;
import org.junit.Test;

public class KarmaTest {
	@Test
	public void testBadKarams() {
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma(" foo++"));
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma("foo--"));
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma("foo++ bar"));
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma("i like foo++ bar"));
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma("/foo++ bar"));
		Assert.assertEquals("foo.bar", KarmaTracker.findThingToKarma("foo.bar++ bar"));
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma("har har foo++ bar++ har har"));

		Assert.assertEquals(null, KarmaTracker.findThingToKarma("foo ++"));

		Assert.assertEquals(null, KarmaTracker.findThingToKarma("++foo"));
	}

	@Test
	@Ignore
	public void testGiveKarma() {
		final String thingToKarma = "blatho";

		KarmaTracker karmaTracker = new KarmaTracker();
		Bot bot = new Bot("mybot", null);
		bot.loadMessagePlugin(karmaTracker);

		Assert.assertEquals(0, karmaTracker.getKarma(thingToKarma));

		Message fooPlusPlus = new Message(bot, "#testChannel", "fred", new MessageParser(thingToKarma + "++"));
		bot.onMockMessage(fooPlusPlus);

		Assert.assertEquals(1, karmaTracker.getKarma(thingToKarma));
	}
}
