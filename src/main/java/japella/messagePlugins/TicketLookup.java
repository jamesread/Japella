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
	private class HttpResponse {
		String content;
		String lastModified;
	}

	private JSONArray jsonOpenTickets;

	private JSONArray jsonNewTickets;
	private HttpResponse httpRespOpenTickets;

	private HttpResponse httpRespNewTickets;

	@Override
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

	@CommandMessage(keyword = "!tickets", target = MessageTarget.CHAT)
	public void onTicketsBrief(Message message) {
		this.recheckTickets();

		if (message.command.hasParam(1)) {
			switch (message.command.getString(1)) {
			case "open":
				message.reply("There are " + this.jsonOpenTickets.length() + " open tickets. Updated: " + this.httpRespOpenTickets.lastModified);
				return;
			case "new":
				message.reply("There are " + this.jsonNewTickets.length() + " new tickets. Updated: " + this.httpRespNewTickets.lastModified);
				return;
			default:
				message.reply("There are " + this.jsonNewTickets.length() + " new tickets and open " + this.jsonOpenTickets.length() + " tickets Updated: " + this.httpRespNewTickets.lastModified);
			}
		}
	}

	@CommandMessage(keyword = "!ticketsfull", target = MessageTarget.CHAT)
	public void onTicketsFull(Message message) {
		this.recheckTickets();

		if (message.command.hasParam(1)) {
			switch (message.command.getString(1)) {
			case "new":
				this.sendTicketList("New", this.jsonNewTickets, message.channel, message.bot);
				return;
			case "open":
				this.sendTicketList("Open", this.jsonOpenTickets, message.channel, message.bot);
				return;
			}
		}
	}

	@Override
	public void onTimerTick(Bot bot, String channel) {
		this.recheckTickets();

		bot.debugMessage("Timer ticking for TicketLookup on " + channel);

		try {
			if (this.jsonNewTickets.length() > 0) {
				bot.sendMessageResponsibly(channel, "There are " + this.jsonNewTickets.length() + " new tickets.");

				for (int i = 0; i < this.jsonNewTickets.length(); i++) {
					JSONObject ticket = this.jsonNewTickets.getJSONObject(i);

					bot.sendMessageResponsibly(channel, " - " + ticket.getInt("id") + " " + ticket.getString("subject") + " https://rt.corp.redhat.com/rt3/Ticket/Display.html?id=" + ticket.getInt("id"));
				}
			}
		} catch (Exception e) {
			bot.sendMessageResponsibly(channel, "Exception: " + e);
			e.printStackTrace();
		}
	}

	public void recheckTickets() {
		try {
			this.httpRespOpenTickets = this.getUrlContent("http://10.33.1.173/rt/openTickets.json");
			this.jsonOpenTickets = new JSONArray(this.httpRespOpenTickets.content);

			this.httpRespNewTickets = this.getUrlContent("http://10.33.1.173/rt/newTickets.json");
			this.jsonNewTickets = new JSONArray(this.httpRespNewTickets.content);
		} catch (Exception e) {
			e.printStackTrace();
		}
	}

	private void sendTicketList(String alias, JSONArray ticketList, String channel, Bot bot) {
		try {
			if (ticketList.length() == 0) {
				bot.sendMessageResponsibly(channel, "There are 0 tickets that are: " + alias);
			} else {
				for (int i = 0; i < ticketList.length(); i++) {
					JSONObject ticket = ticketList.getJSONObject(i);

					String subject = ticket.getString("subject");
					int id = ticket.getInt("id");

					bot.sendMessageResponsibly(channel, alias + " Ticket: #" + id + " - " + subject);
				}
			}
		} catch (Exception e) {
			bot.sendMessageResponsibly(channel, "Oh dear, Exception. :( " + e);
		}
	}
}
