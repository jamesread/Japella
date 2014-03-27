package japellaTest;

import java.util.concurrent.locks.Condition;
import java.util.concurrent.locks.ReentrantLock;

import org.junit.Assert;
import org.junit.Test;

class Producer implements Runnable {
	private final ReentrantLock lock = new ReentrantLock();
	public Condition isProduced = this.lock.newCondition();

	private String productionStatus = "waiting";

	public Producer() {
		new Thread(this).start();
	}

	public String getStatus() throws InterruptedException {
		if (this.productionStatus.equals("waiting")) {
			this.lock.lock();

			this.isProduced.await();

			this.lock.unlock();
		}

		return this.productionStatus;
	}

	@Override
	public void run() {
		this.lock.lock();

		try {
			Thread.sleep(1000);

			this.productionStatus = "Produced!";

			this.isProduced.signalAll();
		} catch (Exception e) {
			System.out.println(e);
		} finally {
			this.lock.unlock();
		}
	}
}

public class TestLocks {
	@Test
	public void testLocks() throws InterruptedException {
		Producer producer = new Producer();

		Assert.assertEquals("Produced!", producer.getStatus());
	}
}
