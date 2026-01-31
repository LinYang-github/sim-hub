import unittest
from unittest.mock import patch, MagicMock
from simhub.client import SimHubClient, SimHubError

class TestSimHubClient(unittest.TestCase):
    def setUp(self):
        self.client = SimHubClient("http://localhost:30030", "test-token")

    @patch('requests.Session.get')
    def test_list_resources(self, mock_get):
        mock_resp = MagicMock()
        mock_resp.ok = True
        mock_resp.json.return_value = {
            "items": [
                {
                    "id": "123",
                    "type_key": "scenario",
                    "name": "test-resource",
                    "owner_id": "admin",
                    "scope": "public"
                }
            ],
            "total": 1
        }
        mock_get.return_value = mock_resp

        res = self.client.list_resources(type_key="scenario")
        self.assertEqual(res.total, 1)
        self.assertEqual(res.items[0].id, "123")
        self.assertEqual(res.items[0].name, "test-resource")

    @patch('requests.Session.get')
    def test_get_resource(self, mock_get):
        mock_resp = MagicMock()
        mock_resp.ok = True
        mock_resp.json.return_value = {
            "id": "123",
            "type_key": "scenario",
            "name": "test-resource",
            "owner_id": "admin",
            "scope": "public"
        }
        mock_get.return_value = mock_resp

        res = self.client.get_resource("123")
        self.assertEqual(res.id, "123")

    @patch('requests.Session.get')
    def test_api_error(self, mock_get):
        mock_resp = MagicMock()
        mock_resp.ok = False
        mock_resp.status_code = 401
        mock_resp.text = "Unauthorized"
        mock_get.return_value = mock_resp

        with self.assertRaises(SimHubError) as cm:
            self.client.get_resource("123")
        self.assertEqual(cm.exception.status_code, 401)

if __name__ == '__main__':
    unittest.main()
