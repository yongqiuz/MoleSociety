import React, { useState, useEffect, useMemo } from 'react';
import { ethers } from 'ethers';

const CONTRACT_ABI = [
  "function mint(uint256 quantity) external payable",
  "function setBaseURI(string baseURI) external",
  "function owner() view returns (address)"
];

// Simplified Pinata upload function (User needs to provide JWT for real usage)
const uploadToPinata = async (file: File, jwt: string) => {
    const formData = new FormData();
    formData.append('file', file);
    
    const metadata = JSON.stringify({
        name: file.name,
    });
    formData.append('pinataMetadata', metadata);

    const options = JSON.stringify({
        cidVersion: 0,
    });
    formData.append('pinataOptions', options);

    try {
        const res = await fetch("https://api.pinata.cloud/pinning/pinFileToIPFS", {
            method: "POST",
            headers: {
                Authorization: `Bearer ${jwt}`,
            },
            body: formData,
        });
        const resData = await res.json();
        return `ipfs://${resData.IpfsHash}`;
    } catch (error) {
        console.error(error);
        throw new Error("Upload failed");
    }
}

export default function SimpleMint() {
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [pinataJwt, setPinataJwt] = useState('');
  const [tokenURI, setTokenURI] = useState('');
  const [contractAddress, setContractAddress] = useState('');
  const [status, setStatus] = useState('');
  const [loading, setLoading] = useState(false);

  const [claimUrl, setClaimUrl] = useState('');

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const contract = params.get('contract') || '';
    const uri = params.get('uri') || '';
    if (contract) {
      setContractAddress(contract);
    }
    if (uri) {
      setTokenURI(uri);
      setStatus('已加载作者分享的 NFT 配置，连接钱包后即可领取');
    }
  }, []);

  useEffect(() => {
    if (contractAddress && tokenURI) {
      const url = `${window.location.origin}/simple-mint?contract=${encodeURIComponent(
        contractAddress
      )}&uri=${encodeURIComponent(tokenURI)}`;
      setClaimUrl(url);
    } else {
      setClaimUrl('');
    }
  }, [contractAddress, tokenURI]);

  const claimQrUrl = useMemo(
    () =>
      claimUrl
        ? `https://api.qrserver.com/v1/create-qr-code/?size=220x220&data=${encodeURIComponent(
            claimUrl
          )}`
        : '',
    [claimUrl]
  );

  const handleImageUpload = async () => {
      if (!imageFile) return alert("Select a file first");
      setLoading(true);
      setStatus("Uploading...");
      
      try {
          // If no JWT provided, use simulation
          let uri = "";
          if (!pinataJwt) {
              await new Promise(r => setTimeout(r, 1000)); // Mock delay
              uri = "ipfs://QmSimulatedHashForDemo_" + Math.floor(Math.random() * 10000);
              setStatus("Simulated Upload Complete (No JWT provided). URI: " + uri);
          } else {
              uri = await uploadToPinata(imageFile, pinataJwt);
              setStatus("Uploaded to IPFS! URI: " + uri);
          }
          setTokenURI(uri);
      } catch (e: any) {
          setStatus("Error: " + e.message);
      } finally {
          setLoading(false);
      }
  };

  const handleSetBaseURI = async () => {
      if (!contractAddress) return alert("Enter contract address");
      if (!(window as any).ethereum) return alert("Install Wallet");
      
      try {
          setLoading(true);
          const provider = new ethers.BrowserProvider((window as any).ethereum);
          const signer = await provider.getSigner();
          const contract = new ethers.Contract(contractAddress, CONTRACT_ABI, signer);
          
          const tx = await contract.setBaseURI(tokenURI);
          setStatus("Tx sent: " + tx.hash);
          await tx.wait();
          setStatus("Base URI updated successfully!");
      } catch (e: any) {
          setStatus("Error: " + e.message);
      } finally {
          setLoading(false);
      }
  };

  const handleMint = async () => {
      if (!contractAddress) return alert("Enter contract address");
      if (!(window as any).ethereum) return alert("Install Wallet");
      
      try {
          setLoading(true);
          const provider = new ethers.BrowserProvider((window as any).ethereum);
          const signer = await provider.getSigner();
          const contract = new ethers.Contract(contractAddress, CONTRACT_ABI, signer);
          
          const tx = await contract.mint(1);
          setStatus("Mint tx sent: " + tx.hash);
          await tx.wait();
          setStatus("NFT Minted Successfully!");
      } catch (e: any) {
          setStatus("Error: " + e.message);
      } finally {
          setLoading(false);
      }
  };

  return (
    <div className="flex flex-col items-center justify-center min-h-[80vh] px-4">
      <div className="w-full max-w-2xl bg-slate-900 p-8 rounded-2xl shadow-xl border border-slate-700">
        <h1 className="text-3xl font-bold mb-8 text-center text-cyan-400">Author Upload & Reader Mint</h1>
        
        <div className="mb-6">
            <label className="block text-sm text-slate-400 mb-1">Contract Address</label>
            <input 
              className="w-full bg-slate-800 border border-slate-600 rounded p-2 text-white"
              value={contractAddress}
              onChange={e => setContractAddress(e.target.value)}
              placeholder="0x..."
            />
        </div>

        <div className="grid md:grid-cols-2 gap-8">
            {/* Author Side */}
            <div className="border border-slate-700 p-4 rounded-xl bg-slate-800/50">
                <h2 className="text-xl font-bold mb-4 text-blue-400">Author</h2>
                
                <div className="mb-4">
                    <label className="block text-xs text-slate-500 mb-1">Pinata JWT (Optional for Real Upload)</label>
                    <input 
                        className="w-full bg-slate-900 border border-slate-700 rounded p-1 text-xs text-white"
                        value={pinataJwt}
                        onChange={e => setPinataJwt(e.target.value)}
                        placeholder="eyJhbGciOiJIUzI1Ni..."
                        type="password"
                    />
                </div>

                <input 
                    type="file" 
                    onChange={e => setImageFile(e.target.files?.[0] || null)}
                    className="mb-4 block w-full text-sm text-slate-400 file:mr-4 file:py-2 file:px-4 file:rounded-full file:border-0 file:text-sm file:font-semibold file:bg-blue-600 file:text-white hover:file:bg-blue-700"
                />

                <button 
                    onClick={handleImageUpload}
                    disabled={loading || !imageFile}
                    className="w-full bg-blue-600 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded disabled:opacity-50 mb-2"
                >
                    1. Upload Image
                </button>

                <button 
                    onClick={handleSetBaseURI}
                    disabled={loading || !tokenURI}
                    className="w-full bg-green-600 hover:bg-green-700 text-white font-bold py-2 px-4 rounded disabled:opacity-50"
                >
                    2. Set Contract URI
                </button>

                {tokenURI && (
                  <>
                    <p className="mt-2 text-xs text-green-400 break-all">URI: {tokenURI}</p>
                    {claimUrl && (
                      <div className="mt-4 space-y-2">
                        <p className="text-xs text-slate-400">将下方二维码印刷或发送给读者扫码领取</p>
                        <div className="flex flex-col items-center gap-3 bg-slate-900/70 rounded-xl p-3 border border-slate-700">
                          {claimQrUrl && (
                            <img
                              src={claimQrUrl}
                              alt="NFT Claim QR"
                              className="w-40 h-40"
                            />
                          )}
                          <p className="text-[10px] text-slate-500 break-all">{claimUrl}</p>
                        </div>
                      </div>
                    )}
                  </>
                )}
            </div>

            {/* Reader Side */}
            <div className="border border-slate-700 p-4 rounded-xl bg-slate-800/50 flex flex-col justify-center">
                <h2 className="text-xl font-bold mb-4 text-purple-400">Reader</h2>
                <p className="text-sm text-slate-400 mb-6">
                    Claim the image uploaded by the author as your own NFT.
                </p>
                <button 
                    onClick={handleMint}
                    disabled={loading}
                    className="w-full bg-purple-600 hover:bg-purple-700 text-white font-bold py-3 px-4 rounded disabled:opacity-50 text-lg"
                >
                    Get This NFT
                </button>
            </div>
        </div>

        {status && (
            <div className="mt-6 p-4 bg-slate-800 rounded border border-slate-600 text-center text-yellow-400">
                {status}
            </div>
        )}
      </div>
    </div>
  );
}
