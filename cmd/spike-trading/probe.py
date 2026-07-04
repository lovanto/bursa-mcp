import cloudscraper
import time

scraper = cloudscraper.create_scraper(browser={
    'browser': 'firefox',
    'platform': 'windows',
    'desktop': True
})

endpoints = [
    "https://idx.co.id/primary/ListedCompany/GetCompanyProfilesDetail?emitenType=s&kodeEmiten=BBCA",
    "https://idx.co.id/primary/ListedCompany/GetFinancialReport?indexFrom=1&pageSize=10&kodeEmiten=BBCA",
    "https://idx.co.id/primary/ListedCompany/GetDividend?kodeEmiten=BBCA",
    "https://idx.co.id/primary/ListedCompany/GetCorporateAction?kodeEmiten=BBCA"
]

for url in endpoints:
    print("\nProbing:", url)
    try:
        res = scraper.get(url, timeout=15)
        print("Status:", res.status_code)
        if res.status_code == 200:
            try:
                data = res.json()
                if isinstance(data, dict):
                    print("Keys:", list(data.keys()))
                    for k in list(data.keys())[:3]:
                        val = data[k]
                        if isinstance(val, list) and len(val) > 0 and isinstance(val[0], dict):
                            print(f"  {k}[0] keys:", list(val[0].keys()))
                elif isinstance(data, list) and len(data) > 0 and isinstance(data[0], dict):
                    print("List of items, first item keys:", list(data[0].keys()))
            except Exception as ex:
                print("Not JSON:", res.text[:100])
        else:
            print("Response:", res.text[:100])
    except Exception as e:
        print("Error:", e)
    
    print("Waiting 15 seconds...")
    time.sleep(15)
