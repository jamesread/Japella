package japella.messagePlugins;

import japella.Bot;
import japella.MessagePlugin;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.net.URL;
import java.net.URLConnection;

import org.json.JSONArray;
import org.json.JSONObject;
 
public class TicketLookup extends MessagePlugin {
	private JSONArray jsonOpenTickets;
	private JSONArray jsonNewTickets; 
	
	private HttpResponse httpRespOpenTickets;
	private HttpResponse httpRespNewTickets; 
	
	private static class HttpResponse {
		String content;
		String lastModified;
	}
	
	public void onTimerTick(Bot bot, String channel) {  
		recheckTickets(); 
		
		System.out.println("Ticking on " + this.getName());
		 
		try {
			if (jsonNewTickets.length() > 0) {  
				bot.sendMessageResponsibly(channel, "There are " + jsonNewTickets.length() + " new tickets.");
				
				for (int i = 0; i < jsonNewTickets.length(); i++) {
					JSONObject ticket = jsonNewTickets.getJSONObject(i);
					  
					bot.sendMessageResponsibly(channel, " - " + ticket.getString("subject"));
				}
			}    
		} catch (Exception e) {
			bot.sendMessageResponsibly(channel, "Exception: " + e);
			e.printStackTrace();
		}
	} 
	
	public String getName() {
		return this.getClass().getSimpleName();
	}
	 
	public HttpResponse getUrlContent(String url) throws Exception {
		URL website = new URL(url);
		HttpResponse httpresp = new HttpResponse();
		
		try {
			URLConnection connection = website.openConnection();
			httpresp.lastModified = connection.getHeaderField("Last-Modified");
			
			BufferedReader br = new BufferedReader(new InputStreamReader(connection.getInputStream()));
			  
			StringBuilder resp = new StringBuilder();
			String input = null; 
			
			while ((input = br.readLine()) != null) {
				resp.append(input);
			}
			 
			br.close();
			
			httpresp.content = resp.toString();
		} catch (IOException e) {
			e.printStackTrace();
			throw new RuntimeException("There was an exception while trying to fetch the URL: " + url + "." + e);
		}
		
		return httpresp;
	}
 
	public void addMessage(String m) {
		 
	} 
	
	public void recheckTickets() {
		try { 
			this.httpRespOpenTickets = getUrlContent("http://10.33.1.189/rt/openTickets.json");
			this.jsonOpenTickets = new JSONArray(httpRespOpenTickets.content); 
			  
			this.httpRespNewTickets = getUrlContent("http://10.33.1.189/rt/newTickets.json");
			this.jsonNewTickets = new JSONArray(httpRespNewTickets.content);
		} catch (Exception e) { 
			e.printStackTrace();
		} 
	}

	public void onMessage(Bot bot, String channel, String sender, String login, String hostname, String message) {
		if (!message.contains("!tickets")) {
			return;
		}  
		
		try {
			recheckTickets(); 
 			
			if (message.equals("!tickets open")) {
				bot.sendMessageResponsibly(channel, "There are " + jsonOpenTickets.length() + " open tickets. Updated: " + httpRespNewTickets.lastModified);				
			} else if (message.equals("!tickets new")) {
				bot.sendMessageResponsibly(channel, "There are " + jsonNewTickets.length() + " open tickets. Updated: " + httpRespOpenTickets.lastModified);
			} else { 
				bot.sendMessageResponsibly(channel, "There are " + jsonOpenTickets.length() + " open tickets and " + jsonNewTickets.length() + " new tickets. Updated: " + httpRespNewTickets.lastModified);
			}
		} catch (Exception e) {
			bot.sendMessageResponsibly(channel, "Sorry, there was an exception: " + e.toString());
		}
	}

	public void onPrivateMessage(Bot bot, String sender, String message) {
		// TODO Auto-generated method stub
		
	}
}
