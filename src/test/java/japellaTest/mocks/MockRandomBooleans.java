package japellaTest.mocks;

import java.util.Random;

public class MockRandomBooleans extends Random {
	public boolean generate = true;

	public MockRandomBooleans(boolean b) {
		this.generate = b;
	}

	@Override
	public boolean nextBoolean() {
		return this.generate;
	}
}
