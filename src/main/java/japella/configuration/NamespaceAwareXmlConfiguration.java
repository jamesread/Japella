package japella.configuration;

import java.io.File;

import javax.xml.parsers.DocumentBuilder;
import javax.xml.parsers.DocumentBuilderFactory;
import javax.xml.parsers.ParserConfigurationException;

import org.apache.commons.configuration.ConfigurationException;
import org.apache.commons.configuration.XMLConfiguration;
import org.apache.commons.configuration.resolver.DefaultEntityResolver;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.xml.sax.SAXException;
import org.xml.sax.SAXParseException;
import org.xml.sax.helpers.DefaultHandler;

public class NamespaceAwareXmlConfiguration extends XMLConfiguration {
	private static final long serialVersionUID = 1L;
	private static final String JAXP_SCHEMA_LANGUAGE = "http://java.sun.com/xml/jaxp/properties/schemaLanguage";
	private static final String W3C_XML_SCHEMA = "http://www.w3.org/2001/XMLSchema";

	private transient DefaultEntityResolver entityResolver = new DefaultEntityResolver();
	private static final transient Logger LOG = LoggerFactory.getLogger(NamespaceAwareXmlConfiguration.class);

	public NamespaceAwareXmlConfiguration(File file) throws ConfigurationException {
		super(file);
	}

	@Override
	protected DocumentBuilder createDocumentBuilder() throws ParserConfigurationException {
		LOG.warn("Creating specialized document builder ");
		if (getDocumentBuilder() != null) {
			return getDocumentBuilder();
		} else {
			DocumentBuilderFactory factory = DocumentBuilderFactory.newInstance();

			if (isValidating()) {
				factory.setValidating(true);
				if (isSchemaValidation()) {
					factory.setNamespaceAware(true);
					factory.setAttribute(JAXP_SCHEMA_LANGUAGE, W3C_XML_SCHEMA);
				}
			}

			factory.setNamespaceAware(true);

			DocumentBuilder result = factory.newDocumentBuilder();

			result.setEntityResolver(entityResolver);

			if (isValidating()) {
				// register an error handler which detects validation errors
				result.setErrorHandler(new DefaultHandler() {
					@Override
					public void error(SAXParseException ex) throws SAXException {
						throw ex;
					}
				});
			}
			return result;
		}
	}
}
