import io.confluent.ksql.api.client.*;
import java.util.concurrent.ExecutionException;

public class DemoInsert {
    public static void main(final String[] args) {
        ClientOptions options = ClientOptions.create()
                .setHost("188.166.225.204")
                .setPort(8088);
        Client client = Client.create(options);
        blockingUsage(client);
        System.out.println("data insert successful");
    }

    private static void blockingUsage(Client client) {
        KsqlObject row = new KsqlObject()
                .put("profileId", "aaaaaaa")
                .put("latitude", 37.4049)
                .put("longitude", -122.0822);

        try {
            client.insertInto("riderLocations",row).get();
        } catch (ExecutionException | InterruptedException e) {
            throw new RuntimeException("Failed to get latest readings", e);
        }
    }
}
