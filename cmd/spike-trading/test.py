import cloudscraper

scraper = cloudscraper.create_scraper(browser={
    'browser': 'chrome',
    'platform': 'windows',
    'desktop': True
})

urls = [
    "https://idx.co.id/primary/ListedCompany/GetTradingInfoSS?code=BBCA&length=30",
    "https://idx.co.id/primary/ListedCompany/GetCompanyProfilesDetail?emitenType=s&kodeEmiten=BBCA",
    "https://idx.co.id/primary/backend/ListedCompany/GetTradingInfoSS?code=BBCA&length=30",
    "https://idx.co.id/primary/backend/ListedCompany/GetCompanyProfilesDetail?emitenType=s&kodeEmiten=BBCA"
]

for url in urls:
    try:
        print("\nHitting:", url)
        res = scraper.get(url, timeout=15)
        print("Status:", res.status_code)
        if res.status_code == 200:
            print("Success!")
            print(res.text[:200])
        else:
            print("Failed!")
            print(res.text[:200])
    except Exception as e:
        print("Error:", e)
