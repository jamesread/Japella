package japella;

import java.util.Timer;
import java.util.TimerTask;

public abstract class MessagePlugin {
	public abstract void addMessage(String m);

	public abstract void onMessage(Bot bot, String channel, String sender, String login, String hostname, String message);
	
	protected static class MessagePluginTimer extends TimerTask {
		private Bot bot;
		private String channel;
		private MessagePlugin plugin;  
		private Timer timer; 
		
		public MessagePluginTimer(Bot bot, String channel, MessagePlugin plugin, int period) {
			this.bot = bot; 
			this.channel = channel;
			this.plugin = plugin;
 			 
			this.timer = new Timer(bot.getName() + " MP timer for " + plugin.getClass().getSimpleName() + " with period of " + period, true);
			
			timer.scheduleAtFixedRate(this, 30, period);  
		}
		
		public void onTimerTick() {
			plugin.onTimerTick(bot, channel);
		}

		@Override
		public void run() {
			onTimerTick();
		} 
	}
	
	public String getName() {
		return this.getClass().getSimpleName();
	}
 
	public abstract void onTimerTick(Bot bot, String channel);
  
	public abstract void onPrivateMessage(Bot bot, String sender, String message);
} 
  