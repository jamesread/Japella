package japella;

public class Server {
	public static class NotFoundException extends Exception {
		private static final long serialVersionUID = 1L;
	} 

	private int port = 0;
	private String name = ""; 
	private String address = "";

	public Server(final String address) {
		this.port = 6667;
		this.name = "???";
		this.address = address;
	}

	public Server(final String name, final String address, final int port) {
		this.port = port;
		this.name = name;
		this.address = address; 
	}

	public String getAddress() {
		return this.address;
	}

	public String getServerName() {
		return this.name;
	}

	public int getPort() {
		return this.port;
	}
}
