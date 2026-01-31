package io.simhub.sdk;

public class SimHubException extends RuntimeException {
    private final int code;

    public SimHubException(String message) {
        this(message, 0);
    }

    public SimHubException(String message, int code) {
        super(message);
        this.code = code;
    }

    public int getCode() {
        return code;
    }
}
