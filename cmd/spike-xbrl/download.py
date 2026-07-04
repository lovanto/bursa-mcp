import cloudscraper
import os

scraper = cloudscraper.create_scraper(browser={'browser': 'firefox', 'platform': 'windows', 'desktop': True})
url = "https://www.idx.co.id/Portals/0/StaticData/ListedCompanies/Corporate_Actions/New_Info_JSX/Jenis_Informasi/01_Laporan_Keuangan/02_Soft_Copy_Laporan_Keuangan//Laporan%20Keuangan%20Tahun%202026/TW1/BBCA/instance.zip"

print("Downloading direct URL:", url)
res = scraper.get(url)
if res.status_code == 200:
    os.makedirs("data", exist_ok=True)
    with open("data/bbca_instance.zip", "wb") as f:
        f.write(res.content)
    print("Success downloading instance zip! Size:", len(res.content))
else:
    print("Failed to download zip:", res.status_code)
    print("Response text:", res.text[:200])
