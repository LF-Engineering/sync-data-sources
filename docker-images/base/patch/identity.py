import requests
from os import getenv

def get_token():
    """Gets access token from SSO for user service.
    """
    task = {
        "grant_type":"client_credentials",
        "client_id": getenv('SSO_API_KEY'),
        "client_secret": getenv('SSO_API_SECRET'),
        "audience": getenv('SSO_AUDIENCE')
    }

    resp = requests.post(getenv('SSO_USER_SERVICE'), json=task)
    if resp.status_code != 200:
        access_token = None
    else:
        access_token = 'Bearer ' + resp.json()['access_token']

    return access_token

def get_email(access_token, username):
    """Makes a request to User service and returns an email
    :param access_token: string from get_token()
    :param username: LFID, from datasource's raw data

    :return string: email or None when username doesn't exist.
    """
    header = {
        'Authorization': access_token,
    }

    if not username:
        return username

    url = getenv('USER_SERVICE_URL') + username
    resp = requests.get(url, headers=header)
    if resp.status_code == 401:
        access_token = get_token()
        resp = requests.get(url, headers=header)

    result = resp.json().get('Data', None)
    if result:
        if len(result) > 0:
            for r in result:
                if r['Username'] == username:
                    for email in r['Emails']:
                        if email['IsPrimary']:
                            email = email.get('EmailAddress', None)
                            return email

emails = {}