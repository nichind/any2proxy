import unittest

from any2proxy import Client


class ClientTest(unittest.TestCase):
    def test_version_and_parse(self):
        client = Client("go", ["run", "../../cmd/a2pd"])
        try:
            self.assertTrue(client.version())
            profile = client.parse("vless://u@example.com:443#one")
            self.assertEqual(profile["nodes"][0]["tag"], "one")
        finally:
            client.close()


if __name__ == "__main__":
    unittest.main()
