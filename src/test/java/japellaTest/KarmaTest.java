package japellaTest;

import japella.messagePlugins.KarmaTracker;

import org.junit.Assert;
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
}
