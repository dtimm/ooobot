# OOOBot

OooBot hosts two API endpoints:

`/v1/outofoffice` - Used to set OOO status.
`/v1/whosout` - Used to get a list of users who are OOO.

The endpoints accept data in the format of URL query parameters.

## Usage
### `/outofoffice`

This command is used to set your OOO status. You can specify a single date or a date range. All dates must be provided in the format `YYYY-MM-DD`.

For a single date:
```bash
/outofoffice [date]
```

For a date range:
```bash
/outofoffice [start] [end]
```
The first date is the start date, and the second date is the end date of your OOO status.

### `/whosout`

This will return the list of everyone who is out of office today.