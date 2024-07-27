1. Proje Yapısı Oluşturma
   Konu: Go projesi ve dizin yapısı oluşturma
   Beklenen Sonuç: Proje dizini, go.mod dosyası, main.go ve raft/raft.go dosyaları
   Not: Go modülünü başlatmayı unutmayın


2. Temel Veri Yapılarını Tanımlama
   Konu: Raft için gerekli struct ve enum tanımlamaları
   Beklenen Sonuç: NodeState enum'ı, LogEntry ve RaftNode struct'ları
   Not: RaftNode struct'ında tüm gerekli alanları eklediğinizden emin olun


3. RaftNode Oluşturma Fonksiyonu
   Konu: NewRaftNode fonksiyonunu yazma
   Beklenen Sonuç: Yeni bir RaftNode örneği oluşturabilen fonksiyon
   Not: Rastgele election timeout süresi belirlemeyi unutmayın


4. Main Fonksiyonu ve İlk Test
   Konu: Main fonksiyonunu yazma ve RaftNode oluşturma
   Beklenen Sonuç: Çalıştırıldığında RaftNode özelliklerini yazdıran program
   Not: Peer listesini ve node ID'sini test için hardcode edebilirsiniz


5. Timer Mekanizmaları
   Konu: Election timer ve heartbeat timer mekanizmalarını ekleme
   Beklenen Sonuç: resetElectionTimer ve startHeartbeat metodları
   Not: Şimdilik sadece log mesajları yazdırın


6. Leader Election Mekanizması
   Konu: Lider seçim sürecini implement etme
   Beklenen Sonuç: startElection metodu ve ilgili yardımcı fonksiyonlar
   Not: RequestVote RPC'sini simüle edin, gerçek network iletişimi henüz gerekmiyor


7. Log Replication Mekanizması
   Konu: Log replikasyon sürecini implement etme
   Beklenen Sonuç: appendEntries ve sendAppendEntries metodları
   Not: AppendEntries RPC'sini simüle edin


8. RPC Handler'ları
   Konu: RequestVote ve AppendEntries RPC handler'larını yazma
   Beklenen Sonuç: RequestVote ve AppendEntries metodları
   Not: Bu aşamada gerçek RPC çağrıları yapmayın, sadece lokal fonksiyon çağrıları kullanın


9. Durum Geçiş Metodları
   Konu: Node durumları arasında geçiş yapan metodları yazma
   Beklenen Sonuç: becomeFollower, becomeCandidate, becomeLeader metodları
   Not: Durum değişikliklerinde gerekli temizleme ve başlatma işlemlerini yapın


10. Safety Kuralları
    Konu: Raft'ın safety kurallarını uygulama
    Beklenen Sonuç: isLogUpToDate metodu ve diğer safety kontrolleri
    Not: Lider seçimi ve log replikasyonunda bu kuralları kullandığınızdan emin olun


11. Basit Network Simülasyonu
    Konu: Node'lar arası iletişimi simüle etme
    Beklenen Sonuç: sendRPC gibi bir yardımcı metod
    Not: Gerçek network yerine channel'lar veya callback'ler kullanabilirsiniz


12. Log Compaction ve Snapshotting
    Konu: Log compaction ve snapshotting mekanizmalarını ekleme
    Beklenen Sonuç: createSnapshot ve applySnapshot metodları
    Not: Basit bir snapshot format'ı kullanın, ileride geliştirilebilir


13. Membership Değişiklikleri
    Konu: Cluster'a node ekleme ve çıkarma işlemlerini implement etme
    Beklenen Sonuç: addServer ve removeServer metodları
    Not: Konfigürasyon değişikliklerini log'a yazmayı unutmayın


14. Hata Yönetimi ve Recovery
    Konu: Node çökmeleri ve network problemlerini ele alma
    Beklenen Sonuç: Hata senaryolarını simüle eden ve recovery sağlayan metodlar
    Not: Çeşitli hata senaryolarını test edin


15. KV Store Entegrasyonu
    Konu: Key-Value Store'u Raft ile entegre etme
    Beklenen Sonuç: Get, Set, Delete operasyonlarını Raft log'una yazan ve uygulayan metodlar
    Not: KV Store'u ayrı bir modül olarak implement edin


16. Test Suite Oluşturma
    Konu: Unit ve integration testleri yazma
    Beklenen Sonuç: Her major fonksiyon için testler ve tam cluster testi
    Not: Edge case'leri ve hata senaryolarını da test etmeyi unutmayın


17. Performans Optimizasyonları
    Konu: Sistemin performansını artıracak optimizasyonlar yapma
    Beklenen Sonuç: Daha hızlı log replikasyonu, daha az RPC çağrısı
    Not: Benchmark testleri yazarak optimizasyonların etkisini ölçün


18. Dökümantasyon ve Örnekler
    Konu: Proje dökümantasyonu ve kullanım örnekleri hazırlama
    Beklenen Sonuç: README dosyası, kod içi yorumlar, örnek kullanım senaryoları
    Not: Projenin nasıl çalıştırılacağını ve kullanılacağını açıkça belirtin

    