package japella.configuration;

import java.io.File;
import java.io.IOException;
import java.util.Vector;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.xml.sax.ErrorHandler;
import org.xml.sax.SAXException;
import org.xml.sax.SAXParseException;

public class ConfigurationErrorHandler implements ErrorHandler {
	private static final transient Logger LOG = LoggerFactory.getLogger(ConfigurationErrorHandler.class);
	private final Vector<SAXParseException> parseErrors = new Vector<SAXParseException>();
	private final File parseSource;

	public ConfigurationErrorHandler(File f) {
		parseSource = f;
	}

	public void assertValidated() throws IOException {
		if (!parseErrors.isEmpty()) {
			throw new IOException("Validation threw " + parseErrors.size() + " parse error(s) in: " + parseSource);
		}
	}

	@Override
	public void error(SAXParseException exception) throws SAXException {
		LOG.info("Configuration parse error (" + exception.getLineNumber() + "): " + exception.getMessage());
		parseErrors.add(exception);
	}

	@Override
	public void fatalError(SAXParseException arg0) throws SAXException {
		LOG.info("Configuration parse fatal error: " + arg0.getMessage());
		parseErrors.add(arg0);
	}

	public Vector<SAXParseException> getParseErrors() {
		return parseErrors;
	}

	public boolean hasErrors() {
		return !parseErrors.isEmpty();
	}

	@Override
	public void warning(SAXParseException arg0) throws SAXException {
		LOG.info("Configuration parse warning: " + arg0.getMessage());
		parseErrors.add(arg0);
	}
}
