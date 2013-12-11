package japella.messagePlugins;

import java.util.Vector;

import japella.Bot;
import japella.MessagePlugin;

public class GaggingPlugin extends MessagePlugin {
	public void addMessage(String m) {
	}   

	public void onMessage(Bot bot, String channel, String sender, String login, String hostname, String message) {
		if (sender.equals(bot.getName())) {
			return;
		}
		
		if (!message.contains(bot.getName())) { 
			return;
		} 
		 
		if (isGagCommand(message)) {
			bot.toggleChannelGag(channel);
		}	 
	}
	
	private static Vector<String> quietCommands = new Vector<String>();
	
	static {
		quietCommands.add("quiet");  
		quietCommands.add("shutup");
		quietCommands.add("silence"); 
		quietCommands.add("pipe down");
	} 
	
	private boolean isGagCommand(String message) {
		for (String gagCommand : quietCommands) { 
			if (message.contains(gagCommand)) {
				return true;  
			}
		} 
		
		return false;
	}

	public String getName() {
		// TODO Auto-generated method stub
		return null;
	}

	public void onTimerTick(Bot bot, String channel) {
		// TODO Auto-generated method stub
		
	}

	public void onPrivateMessage(Bot bot, String sender, String message) {
		// TODO Auto-generated method stub
		
	}

}
