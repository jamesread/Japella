package japellaTest;

import java.io.File;

import javax.xml.XMLConstants;
import javax.xml.transform.dom.DOMSource;
import javax.xml.validation.Schema;
import javax.xml.validation.SchemaFactory;
import javax.xml.validation.Validator;

import org.junit.Test;
import org.w3c.dom.Document;
import org.xml.sax.SAXParseException;

import japella.NamespaceAwareXmlConfiguration;

public class TestXsdReferences {
	@Test
	public void testValidateValidOrder() throws Exception {
		NamespaceAwareXmlConfiguration xmlc = new NamespaceAwareXmlConfiguration(new File("src/test/resources/validClothingOrder.xml"));
		xmlc.setSchemaValidation(true);
   		 
		Document d = xmlc.getDocument();
		SchemaFactory sf = SchemaFactory.newInstance(XMLConstants.W3C_XML_SCHEMA_NS_URI);
		Schema s = sf.newSchema(new File("src/test/resources/clothingOrder.xsd"));
		Validator val = s.newValidator(); 
		val.validate(new DOMSource(d));
	}
	
	@Test(expected=SAXParseException.class)
	public void testValidateNoKey() throws Exception {
		NamespaceAwareXmlConfiguration xmlc = new NamespaceAwareXmlConfiguration(new File("src/test/resources/invalidClothingOrderNoKey.xml"));
		xmlc.setSchemaValidation(true);
   		 
		Document d = xmlc.getDocument();
		SchemaFactory sf = SchemaFactory.newInstance(XMLConstants.W3C_XML_SCHEMA_NS_URI);
		Schema s = sf.newSchema(new File("src/test/resources/clothingOrder.xsd"));
		Validator val = s.newValidator(); 
		val.validate(new DOMSource(d));  
	}
 	  
	@Test(expected=SAXParseException.class)
	public void testValidateInvalidKey() throws Exception {
		NamespaceAwareXmlConfiguration xmlc = new NamespaceAwareXmlConfiguration(new File("src/test/resources/invalidClothingOrderInvalidKey.xml"));
		xmlc.setSchemaValidation(true);
   		 
		Document d = xmlc.getDocument();
		SchemaFactory sf = SchemaFactory.newInstance(XMLConstants.W3C_XML_SCHEMA_NS_URI);
		Schema s = sf.newSchema(new File("src/test/resources/clothingOrder.xsd"));
		Validator val = s.newValidator(); 
		val.validate(new DOMSource(d)); 
	}
}
