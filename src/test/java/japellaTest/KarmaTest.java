package japellaTest;

import japella.messagePlugins.KarmaTracker;

import org.junit.Assert;
import org.junit.Test;

public class KarmaTest {
	@Test
	public void testBadKarams() {
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma("foo++"));
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma("foo--"));
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma("foo++ bar"));
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma("i like foo++ bar"));
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma("_foo++ bar"));
		Assert.assertEquals("foo", KarmaTracker.findThingToKarma("foo.bar++ bar"));

		Assert.assertEquals(null, KarmaTracker.findThingToKarma("foo ++"));
		Assert.assertEquals(null, KarmaTracker.findThingToKarma("_foo_++ bar"));

		Assert.assertEquals(null, KarmaTracker.findThingToKarma("++foo"));
	}
}
