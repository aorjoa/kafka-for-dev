import io.confluent.ksql.api.client.Client;
import io.confluent.ksql.api.client.ClientOptions;
import io.confluent.ksql.api.client.Row;

import java.util.List;
import java.util.concurrent.ExecutionException;

public class DemoQuery {


    public static void main(final String[] args) {
        ClientOptions options = ClientOptions.create()
                .setHost("188.166.225.204")
                .setPort(8088);
        Client client = Client.create(options);

        List<Row> rows = blockingUsage(client);
        for (Row row : rows) {
            System.out.printf("%s : %s\n", row.columnNames(), row.values());
        }
    }

    private static List<Row> blockingUsage(Client client) {
            String pullQuerySql = "SELECT * from ridersNearMountainView WHERE distanceInMiles <= 10;";
            try {
                return client.executeQuery(pullQuerySql).get();
            } catch (ExecutionException | InterruptedException e) {
                throw new RuntimeException("Failed to get latest readings", e);
            }
    }
}
