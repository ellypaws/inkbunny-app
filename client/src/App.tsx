import * as React from "react";
import {useState} from "react";
import Cookies from 'js-cookie';

import {Button} from "@/components/ui/button";
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
    CardFooter,
} from "@/components/ui/card";
import {Input} from "@/components/ui/input";
import {Label} from "@/components/ui/label";

export function LoginForm() {
    const [username, setUsername] = useState('');
    const [password, setPassword] = useState('');

    const handleLogin = async (e: React.FormEvent) => {
        e.preventDefault();
        const response = await fetch('http://localhost:5173/api/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({username, password}),
        });

        if (response.ok) {
            const data = await response.json();
            Cookies.set('PHPSESSID', data.sessionId);
            alert('Login successful!');
            // Proceed with navigating to the next page or showing login success message
        } else {
            alert('Login failed!');
            // Handle login failure (e.g., show error message)
        }
    };

    return (
        <Card className="w-[350px]">
            <CardHeader>
                <CardTitle>Login</CardTitle>
                <CardDescription>Enter your credentials to access your account.</CardDescription>
            </CardHeader>
            <CardContent>
                <form onSubmit={handleLogin}>
                    <div className="grid w-full gap-4">
                        <div className="flex flex-col space-y-1.5">
                            <Label htmlFor="username">Username</Label>
                            <Input id="username" placeholder="guest" value={username}
                                   onChange={(e) => setUsername(e.target.value)}/>
                        </div>
                        <div className="flex flex-col space-y-1.5">
                            <Label htmlFor="password">Password</Label>
                            <Input id="password" type="password" placeholder="Password" value={password}
                                   onChange={(e) => setPassword(e.target.value)}/>
                        </div>
                    </div>
                </form>
            </CardContent>
            <CardFooter className="flex justify-end">
                <Button type="submit" onClick={handleLogin}>Login</Button>
            </CardFooter>
        </Card>
    );
}

export function App() {
    return (
        <div className="flex items-center justify-center h-screen">
            <LoginForm/>
        </div>
    );
}

export default App;
