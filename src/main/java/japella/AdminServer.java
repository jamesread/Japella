package japella;

import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStreamWriter;
import java.net.BindException;
import java.net.ServerSocket;
import java.net.Socket;
import java.util.Vector;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;


public class AdminServer implements Runnable {
	private static class AdminConnection implements Runnable {
		private final Socket s;
		private final Thread runner = new Thread(this, "Admin Connection");

		public AdminConnection(final Socket s) {
			this.s = s; 

			this.runner.start(); 
		}

		public void processCommand(String command) {
			command = command.trim();

			if (command.equals("exit")) {
				Main.instance.shutdown();
			}
		}

		public void run() {
			InputStream is = null;
			OutputStreamWriter osw = null;

			try {
				is = this.s.getInputStream();
				osw = new OutputStreamWriter(this.s.getOutputStream());
			} catch (final Exception e) {
				LOG.error("Could not get proper streams for admin connection.", e);

				return;
			}

			int c;

			try {
				while (true) {
					String buffer = "";
					while ((c = is.read()) != -1) {
						buffer += (char) c;

						if (buffer.length() > 256) {
							LOG.warn("Buffer is getting big. Someone could be trying to hack.");

							buffer = "";   
							this.s.close();

							LOG.warn("IP:" + this.s.getRemoteSocketAddress() + " is trying to fill the buffer.");
							LOG.warn("Closing connection as a precautionary measure.");

							return; 
						}

						if (c == 10) {
							System.out.println("Processing command: " + buffer.toString().trim());
							this.processCommand(buffer.toString());

							buffer = "";
						} else {
							osw.write("Unknown command.\n");
						}
					}
				}
			} catch (final Exception e) { 
				LOG.error("IO error on stream", e);
				e.printStackTrace();
			}  
		}
	}

	private final ServerSocket sock;

	private final Vector<AdminConnection> connections = new Vector<AdminConnection>();
	private final Thread listener = new Thread(this, "Admin Connection Listener");
 
	private boolean listen = true;

	public AdminServer(final int port) throws BindException {
		try {
			this.sock = new ServerSocket(port);
		} catch (final Exception e) {
			LOG.warn("Could not bind admin port: " + port, e);

			throw new BindException();
		}

		this.listener.start();
	}
	
	private static transient final Logger LOG = LoggerFactory.getLogger(AdminServer.class);

	public void run() {
		Socket s;

		while (this.listen) {
			try { 
				s = this.sock.accept();
			} catch (final Exception e) {
				LOG.warn("Unexpected problem while listening for new connecitons.");
				e.printStackTrace();

				continue;
			}
 
			LOG.debug("New admin connection from: " + s.getRemoteSocketAddress());

			synchronized (this.connections) {
				this.connections.add(new AdminConnection(s));
			}
		}
	}
 
	public void stop() throws IOException {
		this.listen = false;
		this.sock.close();
	}
}
