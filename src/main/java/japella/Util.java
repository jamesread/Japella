package japella;

public class Util {
	public static void message(final String type, final String message) {
		System.out.println("[" + type.toUpperCase() + "] " + message);
	}

	public static void messageDebug(final String message) {
		Util.message("DEBUG", message);
	}

	public static void messageError(final String message) {
		Util.message("ERROR", message);

		System.exit(1);
	}

	public static void messageNormal(final String message) {
		Util.message("NORM", message);
	}

	public static void messageWarning(final String message) {
		Util.message("WARN", message);
	}
}
